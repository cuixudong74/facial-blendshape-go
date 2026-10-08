package blendshape

import "testing"

func linTrack(target string, keys ...Keyframe) Track {
	return Track{Target: target, Keys: keys, Interp: InterpLinear}
}

func TestTrackSampleLinear(t *testing.T) {
	tr := linTrack("a", Keyframe{0, 0}, Keyframe{1, 1}, Keyframe{3, 0})
	cases := []struct{ t, want float64 }{
		{-1, 0}, {0, 0}, {0.5, 0.5}, {1, 1}, {2, 0.5}, {3, 0}, {99, 0},
	}
	for _, c := range cases {
		if got := tr.Sample(c.t); !close64(got, c.want) {
			t.Fatalf("Sample(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestTrackSampleStepAndSmooth(t *testing.T) {
	step := Track{Target: "a", Keys: []Keyframe{{0, 0}, {1, 1}}, Interp: InterpStep}
	if got := step.Sample(0.999); got != 0 {
		t.Fatalf("step Sample(0.999) = %v, want 0", got)
	}
	smooth := Track{Target: "a", Keys: []Keyframe{{0, 0}, {1, 1}}, Interp: InterpSmooth}
	if got := smooth.Sample(0.5); !close64(got, 0.5) {
		t.Fatalf("smooth midpoint = %v, want 0.5", got)
	}
	// smooth 在中点两侧对称且端点导数为 0：0.25 处值应小于线性 0.25
	if got := smooth.Sample(0.25); !(got < 0.25 && got > 0) {
		t.Fatalf("smooth(0.25) = %v, want in (0, 0.25)", got)
	}
}

func TestChannelLoopRepeat(t *testing.T) {
	anim := &Animation{Name: "a", Duration: 1.0, Tracks: []Track{
		linTrack("t", Keyframe{0, 0}, Keyframe{1, 1}),
	}}
	c := NewChannel(anim, LoopRepeat)
	c.Advance(1.25) // 进入第 2 周期的 0.25
	out := map[string]float64{}
	c.Evaluate(out)
	if !close64(out["t"], 0.25) {
		t.Fatalf("looped weight = %v, want 0.25", out["t"])
	}
	if c.Done() {
		t.Fatalf("repeat channel must not be done")
	}
}

func TestChannelLoopOnceStopsAtEnd(t *testing.T) {
	anim := &Animation{Name: "a", Duration: 1.0, Tracks: []Track{
		linTrack("t", Keyframe{0, 0}, Keyframe{1, 1}),
	}}
	c := NewChannel(anim, LoopOnce)
	c.Advance(5.0)
	if !c.Done() {
		t.Fatalf("once channel should be done")
	}
	out := map[string]float64{}
	c.Evaluate(out)
	if !close64(out["t"], 1.0) {
		t.Fatalf("final weight = %v, want 1.0", out["t"])
	}
}

func TestChannelPingPong(t *testing.T) {
	anim := &Animation{Name: "a", Duration: 1.0, Tracks: []Track{
		linTrack("t", Keyframe{0, 0}, Keyframe{1, 1}),
	}}
	c := NewChannel(anim, LoopPingPong)
	c.Advance(1.25) // 反向半程：t = 1 - 0.25 = 0.75
	out := map[string]float64{}
	c.Evaluate(out)
	if !close64(out["t"], 0.75) {
		t.Fatalf("pingpong weight = %v, want 0.75", out["t"])
	}
}

// 长时间循环播放：local 时间有界，采样值无累积漂移。
func TestChannelLongRunNoDrift(t *testing.T) {
	anim := &Animation{Name: "a", Duration: 0.5, Tracks: []Track{
		linTrack("t", Keyframe{0, 0}, Keyframe{0.5, 1}),
	}}
	c := NewChannel(anim, LoopRepeat)
	dt := 1.0 / 60.0
	// 模拟约 2.7 小时（60fps 下 585000 帧），落在周期边界附近。
	for i := 0; i < 585000; i++ {
		c.Advance(dt)
	}
	// 585000 * (1/60) ≈ 9750s = 19500 整周期。
	// 关键性质：local 始终有界（落在 [0, Duration) 内），
	// 与理想整周期位置的偏差保持在 1e-9 量级，不随运行时间累积。
	if c.local < 0 || c.local >= anim.Duration {
		t.Fatalf("local %v out of bounds [0, %v)", c.local, anim.Duration)
	}
	deviation := anim.Duration - c.local // 距周期边界的距离（本例在边界前）
	if deviation > 1e-9 && c.local > 1e-9 {
		t.Fatalf("accumulated drift %v exceeds bound after long run", deviation)
	}
	out := map[string]float64{}
	c.Evaluate(out)
	// 采样值应贴近周期边界的理想值（0 或 1），偏差有界。
	w := out["t"]
	if w > 1e-9 && w < 1-1e-9 {
		t.Fatalf("weight drifted to %v, want near 0 or 1", w)
	}
}

// 采样率与时长解耦：同一动画按不同 dt 推进，同一绝对时刻采样一致。
func TestSampleRateDecoupled(t *testing.T) {
	mk := func() *Channel {
		return NewChannel(&Animation{Name: "a", Duration: 2.0, Tracks: []Track{
			linTrack("t", Keyframe{0, 0}, Keyframe{2, 1}),
		}}, LoopRepeat)
	}
	c30, c60 := mk(), mk()
	for i := 0; i < 30; i++ {
		c30.Advance(1.0 / 30.0)
	}
	for i := 0; i < 60; i++ {
		c60.Advance(1.0 / 60.0)
	}
	o30 := map[string]float64{}
	o60 := map[string]float64{}
	c30.Evaluate(o30)
	c60.Evaluate(o60)
	if !close64(o30["t"], o60["t"]) {
		t.Fatalf("30fps=%v vs 60fps=%v", o30["t"], o60["t"])
	}
	if !close64(o30["t"], 0.5) {
		t.Fatalf("weight at t=1s = %v, want 0.5", o30["t"])
	}
}
