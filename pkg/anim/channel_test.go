package anim

import "testing"

func TestChannel_WeightAndOffset(t *testing.T) {
	c := &Clip{Name: "c", Duration: 1, Tracks: []*Track{
		{Target: "x", Keys: []Keyframe{{0, 0}, {1, 10}}},
	}}
	ch := NewChannel(c, 2.0, 0.5, SampleOptions{Interpolation: LinearInterp, Loop: LoopClamp})
	// engine time 3 => local 1 => raw 10 * gain 0.5
	w := ch.WeightsAt(3.0)
	if w["x"] != 5 {
		t.Fatalf("weight = %v want 5", w["x"])
	}
	// before start => local negative, clamp holds first key 0
	w0 := ch.WeightsAt(1.0)
	if w0["x"] != 0 {
		t.Fatalf("before start = %v want 0", w0["x"])
	}
}

func TestChannel_GainClamp(t *testing.T) {
	c := &Clip{Name: "c", Duration: 1, Tracks: []*Track{
		{Target: "x", Keys: []Keyframe{{0, 1}}},
	}}
	if ch := NewChannel(c, 0, 5, DefaultSampleOptions()); ch.Weight != 1 {
		t.Fatalf("gain above 1 not clamped: %v", ch.Weight)
	}
	if ch := NewChannel(c, 0, -1, DefaultSampleOptions()); ch.Weight != 0 {
		t.Fatalf("negative gain not zeroed: %v", ch.Weight)
	}
}
