package anim

import "testing"

func mkTrack() *Track {
	return &Track{Target: "x", Keys: []Keyframe{
		{Time: 0, Value: 0},
		{Time: 1, Value: 10},
		{Time: 2, Value: 0},
	}}
}

func TestSampleTrack_LinearAndStep(t *testing.T) {
	tr := mkTrack()
	if v := SampleTrack(tr, 0.5, SampleOptions{Interpolation: LinearInterp, Loop: LoopRepeat}); v != 5 {
		t.Fatalf("linear mid = %v want 5", v)
	}
	if v := SampleTrack(tr, 0.5, SampleOptions{Interpolation: StepInterp, Loop: LoopRepeat}); v != 0 {
		t.Fatalf("step = %v want 0", v)
	}
	// exact key
	if v := SampleTrack(tr, 1.0, DefaultSampleOptions()); v != 10 {
		t.Fatalf("key = %v want 10", v)
	}
}

func TestSampleTrack_Loops(t *testing.T) {
	tr := mkTrack()
	rep := SampleOptions{Interpolation: LinearInterp, Loop: LoopRepeat}
	// t=2.5 same phase as 0.5
	if v := SampleTrack(tr, 2.5, rep); v != SampleTrack(tr, 0.5, rep) {
		t.Fatal("repeat loop not periodic")
	}
	// many cycles
	if v := SampleTrack(tr, 1000.5, rep); v != 5 {
		t.Fatalf("t=1000.5 = %v want 5", v)
	}
	pp := SampleOptions{Interpolation: LinearInterp, Loop: LoopPingPong}
	// ping-pong: t=2.5 plays backward, phase 1.5 => value 5
	if v := SampleTrack(tr, 2.5, pp); v != 5 {
		t.Fatalf("pingpong t=2.5 = %v want 5", v)
	}
	clamp := SampleOptions{Interpolation: LinearInterp, Loop: LoopClamp}
	if v := SampleTrack(tr, 100, clamp); v != 0 {
		t.Fatalf("clamp end = %v want 0", v)
	}
	if v := SampleTrack(tr, -5, clamp); v != 0 {
		t.Fatalf("clamp start = %v want 0", v)
	}
}

func TestSampleTrack_NoAccumulatedDrift(t *testing.T) {
	// Absolute sampling of a huge t must equal sampling its phase directly.
	tr := mkTrack()
	opts := DefaultSampleOptions()
	huge := 86400.0 * 30 // 30 days of seconds
	direct := SampleTrack(tr, 0.25, opts)
	got := SampleTrack(tr, huge+0.25, opts)
	// modulo float precision tolerance
	if d := got - direct; d > 1e-9 || d < -1e-9 {
		t.Fatalf("drift after 30 days: got %v want %v", got, direct)
	}
}

func TestSampleClip_Deterministic(t *testing.T) {
	c := &Clip{Name: "c", Duration: 2, Tracks: []*Track{
		{Target: "z", Keys: []Keyframe{{0, 0}, {1, 1}}},
		{Target: "a", Keys: []Keyframe{{0, 0}, {1, 2}}},
	}}
	w1 := SampleClip(c, 1, DefaultSampleOptions())
	w2 := SampleClip(c, 1, DefaultSampleOptions())
	if w1["a"] != w2["a"] || w1["z"] != w2["z"] {
		t.Fatal("clip sampling not reproducible")
	}
}

func TestClipValidate(t *testing.T) {
	bad := &Clip{Name: "x", Duration: 1, Tracks: []*Track{
		{Target: "t", Keys: []Keyframe{{Time: 1}, {Time: 0}}},
	}}
	if err := bad.Validate(); err == nil {
		t.Fatal("unsorted keys must fail validation")
	}
	good := &Clip{Name: "x", Duration: 1, Tracks: []*Track{
		{Target: "t", Keys: []Keyframe{{0, 0}, {1, 1}}},
	}}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid clip rejected: %v", err)
	}
}

func TestSampleTrack_EdgeCases(t *testing.T) {
	if v := SampleTrack(nil, 1, DefaultSampleOptions()); v != 0 {
		t.Fatalf("nil track = %v want 0", v)
	}
	single := &Track{Target: "s", Keys: []Keyframe{{Time: 2, Value: 7}}}
	if v := SampleTrack(single, 100, SampleOptions{Interpolation: LinearInterp, Loop: LoopClamp}); v != 7 {
		t.Fatalf("single key clamp = %v want 7", v)
	}
	if v := SampleTrack(single, 0, DefaultSampleOptions()); v != 7 {
		t.Fatalf("single key repeat = %v want 7", v)
	}
	// negative time repeat wraps correctly
	tr := mkTrack()
	v := SampleTrack(tr, -0.5, SampleOptions{Interpolation: LinearInterp, Loop: LoopRepeat})
	if v != SampleTrack(tr, 1.5, SampleOptions{Interpolation: LinearInterp, Loop: LoopRepeat}) {
		t.Fatalf("negative phase mismatch: %v", v)
	}
	// empty clip sample is empty
	if got := SampleClip(&Clip{Name: "e", Duration: 1}, 0.5, DefaultSampleOptions()); len(got) != 0 {
		t.Fatalf("empty clip produced weights %v", got)
	}
}

func TestSampleTrack_NilChannel(t *testing.T) {
	var ch *Channel
	if w := ch.WeightsAt(1); len(w) != 0 {
		t.Fatalf("nil channel = %v", w)
	}
}
