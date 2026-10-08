package blendshape

// Channel 为一路动画通道：一个动画的播放实例。
// 多通道并发驱动时，各通道权重按通道权重缩放后合并（同名目标取最大）。
//
// 为消除长时间运行的累积误差，播放位置用 (cycles, local) 表示：
// local 始终落在 [0, Duration) 内，浮点误差不会随运行时间无限累积。
type Channel struct {
	Anim   *Animation
	Loop   LoopMode
	Weight float64 // 通道整体权重，钳制到 [0,1]

	cycles  uint64
	local   float64
	playing bool
	done    bool
}

// NewChannel 创建通道并开始播放。
func NewChannel(anim *Animation, loop LoopMode) *Channel {
	return &Channel{Anim: anim, Loop: loop, Weight: 1, playing: true}
}

// Advance 推进通道时间。dt 为负或 NaN 时忽略。
func (c *Channel) Advance(dt float64) {
	if !c.playing || c.done || dt <= 0 || dt != dt {
		return
	}
	d := c.Anim.Duration
	if d <= 0 {
		if c.Loop == LoopOnce {
			c.done = true
		}
		return
	}
	c.local += dt
	switch c.Loop {
	case LoopOnce:
		if c.local >= d {
			c.local = d
			c.done = true
			c.playing = false
		}
	case LoopRepeat:
		for c.local >= d { // 整周期进位，local 保持有界
			c.local -= d
			c.cycles++
		}
	case LoopPingPong:
		for c.local >= d {
			c.local -= d
			c.cycles++
		}
	}
}

// Done 报告 LoopOnce 通道是否已播放完毕。
func (c *Channel) Done() bool { return c.done }

// Time 返回当前有效采样时间（PingPong 奇数周期反向）。
func (c *Channel) Time() float64 {
	if c.Loop == LoopPingPong && c.cycles%2 == 1 {
		return c.Anim.Duration - c.local
	}
	return c.local
}

// Evaluate 采样当前时刻所有轨道，向 out 合并目标权重（已乘通道权重）。
// 同名目标取最大值，保证多通道合并结果与通道求值顺序无关。
func (c *Channel) Evaluate(out map[string]float64) {
	w := clamp01(c.Weight)
	if w == 0 {
		return
	}
	t := c.Time()
	for i := range c.Anim.Tracks {
		tr := &c.Anim.Tracks[i]
		v := tr.Sample(t) * w
		if v > out[tr.Target] {
			out[tr.Target] = v
		}
	}
}
