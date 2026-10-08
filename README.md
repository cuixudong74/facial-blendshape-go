# facial-blendshape-go

面部形变目标（blendshape / morph target）驱动的表情动画运行时。
将表情控制从硬编码收敛为**可配置、可热更新**的能力：形变目标即数据，
运行时负责加权混合、分层组合、关键帧动画驱动、热加载与并发求值。

## 形变语义

- **Mesh**：基础网格，`Positions` 为 `[x0 y0 z0 x1 y1 z1 ...]`，运行期只读。
- **MorphTarget**：一个形变目标，`Deltas` 为相对基础网格的逐顶点位移增量，
  长度 `3*VertexCount`。加载后不可变，替换即整体换版本。
- 最终顶点：`out = base + Σ wᵢ·deltaᵢ`，按层分阶段计算（见下）。

### 分层组合（顺序固定，结果与组合顺序无关）

| 层 | 权重处理 | 应用时机 |
|---|---|---|
| `LayerBase` 基础层 | 按策略归一化（见下） | 第 1 步，混合到基础网格 |
| `LayerAdditive` 叠加层 | 钳制到 [0,1]，不归一化 | 第 2 步，在基础层结果上直接叠加 |
| `LayerCorrective` 校正层 | 钳制到 [0,1] × 各 `Drivers` 目标（基础层归一化后）权重乘积 | 第 3 步，最后应用 |

层内按目标名排序遍历，浮点累加顺序确定，**给定输入下输出逐位可复现**。
校正目标用于修正基础/叠加组合产生的伪影（in-between corrective）；
其驱动目标缺席（权重为 0）时校正自动不生效。

### 权重归一化策略（`NormalizePolicy`）

1. `NaN` / `±Inf` → 0；
2. 负权重 → 0；
3. 权重和超过 `MaxSum`（默认 1.0，`<=0` 表示不限制）→ 按比例整体缩放。

求和与缩放均按键序进行，结果确定。被修正的键会返回以便上层告警。
非法权重不会中断整帧，只影响对应目标。

## 关键帧动画

- `Track`：单目标权重轨道，关键帧按时间升序；支持 `InterpStep` /
  `InterpLinear` / `InterpSmooth`（smoothstep）。
- **采样率与时长解耦**：`Track.Sample(t)` 可在任意时刻取值（二分查找区间），
  30/60/任意 fps 驱动同一动画结果一致（见 `TestSampleRateDecoupled`）。
- `Channel`：动画播放实例，支持 `LoopOnce` / `LoopRepeat` / `LoopPingPong`。
  播放位置以 `(cycles, local)` 表示，`local` 恒在 `[0, Duration)` 内，
  **长时间运行误差有界不累积**（`TestChannelLongRunNoDrift` 模拟 60fps
  约 2.7 小时后偏差仍在 1e-9 量级）。
- 多通道合并：各通道权重乘以通道权重后，同名目标取最大值，
  合并结果与通道顺序无关。

## 热加载与热替换

`Registry` 采用**写时复制 + 原子指针**：

- 写路径（`Load` / `Replace` / `Remove`）在互斥锁内先完整校验
  （维度匹配、无 NaN/Inf），构造完整新版本后一次性原子发布，版本号单调递增；
- 读路径无锁：`Snapshot()` 取得不可变版本，整帧持有。
  **替换期间正在驱动的动画继续读旧版本，任一帧只会看到某个已完整加载的版本**，
  绝不读到半成品（`TestRegistryHotSwapConsistency`、
  `TestEngineEvaluateDuringHotSwap` 在 `-race` 下验证）。

形变数据因此可以放在配置文件 / 资源包中，运行时替换，无需重新发版。

## 并发模型

- 基础网格与已发布目标数据只读，多 goroutine 共享无数据竞争；
- `Engine.Advance` / `Engine.Evaluate` 内部对共享通道状态加锁
  （锁内只做 O(轨道数) 的权重采样），逐顶点混合在锁外并发执行；
- 每个 goroutine 使用自己的输出缓冲，**不同并发度下输出一致**
  （`TestEngineConcurrentEvaluateConsistent`）。

## 异常链路

| 异常 | 策略 |
|---|---|
| 权重 NaN/Inf/负值 | 置 0，计入修正列表 |
| 权重和超上限 | 按比例缩放 |
| 目标维度不一致 / 数据含 NaN | 加载时拒绝（`Registry`）；评估时跳过该目标 |
| 权重引用了不存在的目标 | 跳过该目标 |
| 任一跳过 | 记录 `Issue{Target, Reason}`，可定位到具体目标，**整帧不崩溃** |

## 性能

- 每帧成本 **O(活跃目标数 × 顶点数)**，与注册表中目标总数无关；
  权重为 0 的目标不参与计算。
- 输出写入调用方持有的缓冲，跨帧复用，**不做每帧整份网格拷贝**
  （仅一次 `base` 拷贝作为累加起点，这本身就是输出）。
- 每帧堆分配为小常量（权重合并用的几个 map/slice），不随顶点数增长。
- 基准（Xeon 8255C，`go test -bench .`）：

| 顶点数 | 活跃目标 | 耗时/帧 |
|---|---|---|
| 1 万 | 4 | ~0.24 ms |
| 1 万 | 32 | ~3.2 ms |
| 5 万 | 32 | ~20 ms |

耗时随活跃目标数近似线性增长，符合复杂度预期。

## 使用示例

```go
mesh := &blendshape.Mesh{Positions: pos, VertexCount: n}
e := blendshape.NewEngine(mesh)

// 加载形变目标（可来自配置/资源包，运行时可 Replace 热更新）
_ = e.Registry().Load(&blendshape.MorphTarget{
    Name: "smile", Layer: blendshape.LayerBase,
    Deltas: smileDeltas, VertexCount: n,
})

// 关键帧动画通道
anim := &blendshape.Animation{
    Name: "smile_loop", Duration: 1.0,
    Tracks: []blendshape.Track{{
        Target: "smile", Interp: blendshape.InterpSmooth,
        Keys: []blendshape.Keyframe{{0, 0}, {0.5, 1}, {1, 0}},
    }},
}
e.AddChannel(blendshape.NewChannel(anim, blendshape.LoopRepeat))

out := make([]float32, 3*n) // 跨帧复用
for {
    e.Advance(1.0 / 60.0)
    if issues := e.Evaluate(out); len(issues) > 0 {
        log.Println(issues) // 定位到具体目标的跳过原因
    }
    render(out)
}
```

## 测试

```sh
go test -race ./...   # 全部单测（含并发与热替换，竞态检测）
go test -bench . ./... # 性能基准
```

覆盖：权重归一化（非法值/超和/确定性）、分层校正（顺序/驱动乘积/损坏跳过）、
关键帧插值（三种策略/循环/长跑无漂移/采样率解耦）、热替换（版本一致/无撕裂）、
并发（多 goroutine 输出一致）、损坏数据（维度不符/NaN/缺失目标）。
