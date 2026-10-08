package blendshape

import "sort"

// Interp 为关键帧插值策略。
type Interp int

const (
	InterpStep   Interp = iota // 阶梯：取左侧关键帧值
	InterpLinear               // 线性插值
	InterpSmooth               // smoothstep 平滑插值
)

// LoopMode 为循环播放模式。
type LoopMode int

const (
	LoopOnce     LoopMode = iota // 播放一次后停在末帧
	LoopRepeat                   // 循环
	LoopPingPong                 // 往返
)

// Keyframe 为一个关键帧。
type Keyframe struct {
	Time  float64 // 秒
	Value float64
}

// Track 为单个形变目标的权重轨道。采样率与时长解耦：
// 采样按任意时间点取值，与关键帧分布无关。
type Track struct {
	Target string
	Keys   []Keyframe // 必须按 Time 升序
	Interp Interp
}

// Sample 在任意时间 t 采样轨道值。t 越界时钳制到端点值。
// 二分查找定位区间，给定 t 输出唯一确定。
func (tr *Track) Sample(t float64) float64 {
	n := len(tr.Keys)
	if n == 0 {
		return 0
	}
	if t <= tr.Keys[0].Time {
		return tr.Keys[0].Value
	}
	if t >= tr.Keys[n-1].Time {
		return tr.Keys[n-1].Value
	}
	// 找最大的 i 使 Keys[i].Time <= t。
	i := sort.Search(n, func(i int) bool { return tr.Keys[i].Time > t }) - 1
	k0, k1 := tr.Keys[i], tr.Keys[i+1]
	span := k1.Time - k0.Time
	if span <= 0 {
		return k0.Value
	}
	u := (t - k0.Time) / span
	switch tr.Interp {
	case InterpStep:
		return k0.Value
	case InterpSmooth:
		u = u * u * (3 - 2*u)
		fallthrough
	default: // InterpLinear
		return k0.Value + (k1.Value-k0.Value)*u
	}
}

// Animation 为一条关键帧动画，包含多条轨道。
type Animation struct {
	Name     string
	Duration float64 // 秒；通道时长与采样率解耦
	Tracks   []Track
}
