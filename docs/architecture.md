# 架构与数据流

```
                 ┌──────────────────────────────────────────────┐
 JSON 目标文件 ──▶│ store.Loader (解码/checksum/维度校验)          │
 *.json          └───────────────┬──────────────────────────────┘
                                 │ 整批校验通过
                                 ▼
                    ┌────────────────────────┐
  Publish/Replace ─▶│ Store                  │  atomic.Pointer[Snapshot]
  DirectoryReloader │ Snapshot{version,map}  │◀─ 不可变、深拷贝、单调版本
                    └───────────┬────────────┘
                                │ Current()：每帧只取一次
                                ▼
 config.json ─▶ WeightPolicy ┌─────────────┐
                             │ Engine      │
 Clip(s) ─▶ Channel(s) ─────▶│ 合并通道权重 │
 (anim: 插值/循环/绝对时间)   │ 分层求值     │──▶ Frame{Vertices,Version,
                             └─────────────┘                 Issues,Errors}
                                ▲
                      多 goroutine 并发调用（无共享可变状态）
```

## 每帧数据流（确定性保证）

1. `snap = store.Current()`：原子加载一次不可变快照。
2. `MergeChannelWeights(channels, t)`：
   - 每条通道把绝对时间 `t - StartTime` 送入纯函数 `SampleClip`（不增量累加）；
   - 通道按切片顺序、轨按目标名顺序，相加得到原始权重。
3. `LayerMixer.Evaluate(targets, rawWeights, policy, out)`：
   - 校验每个目标（失败记录到 `TargetError{Target,Op,Reason}` 并跳过）；
   - 按 kind 分三层；base 层归一化，additive/corrective 保持独立强度；
   - 层内按 `(Priority, Name)` 排序后累加，求和顺序固定；
   - 结果写入调用方复用的 `out`。
4. 返回 `Frame`，并标注它所属的快照 `Version`。

## 不可变与所有权

- `Store.commit` 对每个目标 `Clone()`：发布后调用方再改自己的切片不影响在驱帧。
- `Snapshot.targets` 在发布后不再写入；读取无锁（atomic pointer + 只读 map）。
- 引擎里唯一的派生缓存（版本→有序目标切片）由 RWMutex 保护，且只在
  版本推进时整体替换。

## 异常链路（不崩溃策略）

| 异常 | 处理 | 可定位信息 |
|------|------|-----------|
| 权重 NaN/Inf | 跳过或置 0（由 `SkipInvalid` 定） | `WeightIssue.Target/Value/Reason` |
| 权重超界 | 夹紧或丢弃（由 `Clamp` 定） | 同上 |
| 未知目标名 | 忽略 | `WeightIssue.Reason="unknown target"` |
| 目标维度不符/含 NaN/未知 kind | 该目标跳过，其余照常；发布期则整批拒绝 | `TargetError.Target/Op/Reason` |
| JSON 损坏 | 该文件跳过，其余加载；热更扫描则保留旧快照 | 以文件名/目标名上报 |
| checksum 不符 | 拒绝该目标（疑似篡改/损坏） | 声明值 vs 实际值 |

一帧的求值永远返回结果；所有问题都聚合在 `Frame.Issues / Frame.Errors`，
调用方决定记录、告警还是忽略。
