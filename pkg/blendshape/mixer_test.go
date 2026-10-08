package blendshape

import (
	"math"
	"testing"
)

func mkNeutral(n int) []Vec3 {
	v := make([]Vec3, n)
	for i := range v {
		v[i] = Vec3{X: float64(i), Y: 0, Z: 0}
	}
	return v
}

func mkTarget(name string, kind TargetKind, prio int, n int, d Vec3) *Target {
	deltas := make([]Vec3, n)
	for i := range deltas {
		deltas[i] = d
	}
	return &Target{Name: name, Kind: kind, Priority: prio, VertexCount: n, Deltas: deltas}
}

func TestMix_BasicBlend(t *testing.T) {
	neutral := mkNeutral(2)
	m := NewMixer(neutral)
	tg := mkTarget("smile", KindBase, 0, 2, Vec3{X: 0, Y: 1, Z: 0})

	out, issues, errs := m.Mix([]*Target{tg}, map[string]float64{"smile": 1},
		WeightPolicy{Min: 0, Max: 1, Clamp: true, Normalize: true, SkipInvalid: true}, nil)
	if len(issues) != 0 || len(errs) != 0 {
		t.Fatalf("issues=%v errs=%v", issues, errs)
	}
	if out[0].Y != 1 {
		t.Fatalf("got %v want Y=1", out[0])
	}
	// neutral X preserved
	if out[1].X != 1 {
		t.Fatalf("neutral X lost: %v", out[1])
	}
}

func TestMix_OrderIndependent(t *testing.T) {
	neutral := mkNeutral(1)
	a := mkTarget("a", KindBase, 0, 1, Vec3{X: 0.1, Y: 0, Z: 0})
	b := mkTarget("b", KindBase, 0, 1, Vec3{X: 0.7, Y: 0, Z: 0})
	p := WeightPolicy{Min: 0, Max: 1, Clamp: true, Normalize: true, SkipInvalid: true}

	m := NewMixer(neutral)
	o1, _, _ := m.Mix([]*Target{a, b}, map[string]float64{"a": 0.3, "b": 0.7}, p, nil)
	o2, _, _ := m.Mix([]*Target{b, a}, map[string]float64{"a": 0.3, "b": 0.7}, p, nil)
	if o1[0] != o2[0] {
		t.Fatalf("order dependent: %v vs %v", o1[0], o2[0])
	}
}

func TestMix_BadTargetSkipped(t *testing.T) {
	neutral := mkNeutral(2)
	m := NewMixer(neutral)
	good := mkTarget("ok", KindBase, 0, 2, Vec3{Y: 1})
	bad := mkTarget("bad", KindBase, 0, 3, Vec3{}) // vertex count mismatch

	out, _, errs := m.Mix([]*Target{good, bad}, map[string]float64{"ok": 1, "bad": 1},
		DefaultWeightPolicy(), nil)
	if len(errs) != 1 {
		t.Fatalf("want 1 target error, got %d", len(errs))
	}
	if te, ok := errs[0].(*TargetError); !ok || te.Target != "bad" {
		t.Fatalf("error must locate bad target: %v", errs[0])
	}
	if out[0].Y != 1 {
		t.Fatalf("good target must still apply, got %v", out[0])
	}
}

func TestMix_UnknownWeightReported(t *testing.T) {
	neutral := mkNeutral(1)
	m := NewMixer(neutral)
	a := mkTarget("a", KindBase, 0, 1, Vec3{Y: 1})
	_, issues, _ := m.Mix([]*Target{a}, map[string]float64{"ghost": 1}, DefaultWeightPolicy(), nil)
	if len(issues) != 1 || issues[0].Target != "ghost" {
		t.Fatalf("want issue for ghost, got %v", issues)
	}
}

func TestMix_NoMeshCopyBeyondOutput(t *testing.T) {
	neutral := mkNeutral(4)
	m := NewMixer(neutral)
	a := mkTarget("a", KindBase, 0, 4, Vec3{Y: 1})
	out := make([]Vec3, 4)
	ret, _, _ := m.Mix([]*Target{a}, map[string]float64{"a": 1}, DefaultWeightPolicy(), out)
	if &ret[0] != &out[0] {
		t.Fatal("mixer must reuse provided output buffer")
	}
	// neutral of mixer untouched after two frames
	if m.neutral[3].X != 3 {
		t.Fatal("neutral mesh mutated")
	}
}

func TestTargetValidate_NilAndBadKind(t *testing.T) {
	var nilT *Target
	if err := nilT.Validate(2); err == nil {
		t.Fatal("nil target must fail")
	}
	bad := &Target{Name: "x", Kind: KindUnknown, VertexCount: 1, Deltas: []Vec3{{}}}
	if err := bad.Validate(1); err == nil {
		t.Fatal("unknown kind must fail")
	}
	nan := &Target{Name: "x", Kind: KindBase, VertexCount: 1, Deltas: []Vec3{{X: math.NaN()}}}
	if err := nan.Validate(1); err == nil {
		t.Fatal("NaN delta must fail")
	}
	if c := nan.Clone(); c == nil || c.Name != "x" {
		t.Fatal("clone wrong")
	}
	if c := nilT.Clone(); c != nil {
		t.Fatal("nil clone must be nil")
	}
}
