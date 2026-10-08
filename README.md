# Facial Blendshape Runtime

一套“面部形变目标（blendshape / morph target）”驱动的表情动画运行时。
目标：把过去写死在代码里、叠加顺序不确定、换数据必须发版的表情结果，
收敛成**可配置、可热更新、可复现、并发安全**的一套能力。

## 解决的问题与对应能力

| # | 需求 | 实现位置 |
|---|------|----------|
| 1 | 多路加权混合、权重归一化、非法/超范围处理、结果可复现 | `pkg/blendshape/weights.go`, `mixer.go` |
| 2 | 基础/叠加/校正三层，层内固定优先级顺序 | `pkg/blendshape/layer.go` |
| 3 | 关键帧动画、插值策略、循环、时长与采样率解耦、长时无漂移 | `pkg/anim/sampler.go`, `channel.go` |
| 4 | 运行时热加载/替换，替换期间不读到半成品 | `pkg/store/store.go`, `reloader.go` |
| 5 | 多通道/多目标并发驱动，无数据竞争，输出一致 | `pkg/engine/engine.go` |
| 6 | 数据损坏/维度不一致/非法权重不崩溃，定位到具体目标 | `pkg/blendshape/errors.go`, `pkg/store/loader.go` |
| 7 | 开销可控、不每帧拷贝整网格，附基准 | `O(V·T)` + benchmark，见 docs/performance.md |
| 8 | 自动化测试与文档 | 各包 `*_test.go`、`docs/` |

## 核心概念（形变语义）

- **Neutral（中性网格）**：V 个顶点的基准位置，运行时只读。
- **Target（形变目标）**：每个顶点一个 delta `Vec3{X,Y,Z}`，长度必须等于 V。
  发布后不可变（store 会深拷贝）。
- **Weight（权重）**：`target -> float64`。原始权重先经 `WeightPolicy` 处理，
  再参与混合。
- **Layer（层）**，求值顺序固定：
  1. `base`（基础）：层内权重 clamp 后**归一化为和 1**，表达主动表情；
  2. `additive`（叠加）：各自独立权重（clamp，**不归一化**），叠在基础之上；
  3. `corrective`（校正）：最后施加，用于修正“多个表情同时出现”产生的穿插。
  同一层内严格按 `Priority` 升序、再按 `Name` 字典序施加——浮点求和顺序确定，
  因此输入顺序不同不会导致结果差异。

最终顶点：
`out = neutral + Σ_base wᵢ·dᵢ + Σ_add wⱼ·dⱼ + Σ_corr wₖ·dₖ`

## 权重策略 WeightPolicy

```go
type WeightPolicy struct {
    Min, Max    float64 // 合法区间（默认 [0,1]）
    Clamp       bool    // 超界是否夹紧；false 则丢弃该权重
    Normalize   bool    // base 层是否归一化为和 1
    SkipInvalid bool    // NaN/Inf 是跳过(true)还是按 0 处理(false)
}
```

- NaN / +Inf / -Inf 恒为非法；
- 超界：`Clamp=true` 夹紧并在 `WeightIssue` 中记录，否则丢弃；
- 归一化只作用于 **base 层**（叠加/校正保持独立强度）；
- 所有重排按目标名字典序做，**相同输入产出逐位相同**的结果。
- 无法识别的目标名不会崩溃，会进入 `WeightIssue{Target, Value, Reason}`。

## 动画语义

- `Clip` = 若干 `Track`，一条 Track 驱动一个目标，由有序 `Keyframe{Time,Value}` 组成。
- **时长是 Clip 的属性；采样是连续时间函数**，不存在固定采样率，因此帧率任意、
  可变帧率都安全。
- 插值：`LinearInterp` / `StepInterp`。
- 循环：`LoopRepeat`（取模）、`LoopPingPong`（乒乓）、`LoopClamp`（端点保持）。
- **无累积误差**：采样永远对“绝对引擎时间”做纯函数映射
  （`mapTime(t)`），绝不增量累加相位。连续运行 30 天与直接采样对应相位一致
  （测试 `TestSampleTrack_NoAccumulatedDrift`）。
- `Channel` 是 Clip 的一次播放实例：`StartTime` + `Weight`（通道增益）。
  多个通道驱动同一目标时**相加**，通道顺序由调用方的切片顺序决定。

## 热更新语义（重要）

- `Store` 用 `atomic.Pointer[Snapshot]` 发布**不可变快照**，版本号单调递增。
- 一次替换 = 私有地解码 → 深拷贝 → 校验 → 全部成功后**一次原子 Store**。
- 任何一个目标损坏/维度不符/重名，**整批拒绝**，继续服务旧版本。
- 引擎每帧只 `Load()` 一次快照，因此一帧要么完整是旧版，要么完整是新版，
  绝不会出现“一半新一半旧”的撕裂帧（测试
  `TestEngine_FrameNeverTornDuringHotReplace`、`TestReaderNeverTorn`）。
- `DirectoryReloader` 轮询目录下 `*.json`，按内容 checksum 判断是否变化；
  目录中出现损坏文件时本次扫描整体放弃，保留上一良好版本。

## 并发语义

- Neutral、Target、Snapshot 均只读；引擎求值无可变每帧状态。
- `LayerMixer`/`Mixer` 跨调用无状态，可被任意 goroutine 共享。
- 相同输入在不同并发度下**逐位一致**（`-race` 下测试
  `TestEngine_ConcurrentConsistency`）。

## 快速开始

```go
st := store.NewStore()
st.Publish(targets, vertexCount)                       // 发布目标
eng := engine.NewEngine(neutral, blendshape.DefaultWeightPolicy(), st)

ch := anim.NewChannel(clip, startTime, gain, anim.DefaultSampleOptions())
frame := eng.EvaluateAt([]*anim.Channel{ch}, engineTime, reusableOut)
// frame.Vertices / frame.Version / frame.Issues / frame.Errors
```

配置文件（JSON）格式见 `docs/config.md`。可运行演示见 `cmd/demo`。

## 测试与基准

```bash
go test ./... -race -count=1
go test ./pkg/blendshape/ -bench . -benchmem
go test ./pkg/engine/   -bench . -benchmem
```

复杂度与基准数字见 `docs/performance.md`。
