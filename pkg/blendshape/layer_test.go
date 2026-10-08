package blendshape

import "testing"

func TestLayer_BaseAdditiveCorrectiveOrder(t *testing.T) {
	neutral := mkNeutral(1)
	base := mkTarget("b1", KindBase, 0, 1, Vec3{X: 1})
	add := mkTarget("a1", KindAdditive, 0, 1, Vec3{X: 2})
	corr := mkTarget("c1", KindCorrective, 0, 1, Vec3{X: -0.5})
	m := NewLayerMixer(neutral)

	out, _, errs := m.Evaluate([]*Target{corr, add, base}, // shuffled input
		map[string]float64{"b1": 1, "a1": 1, "c1": 1}, DefaultWeightPolicy(), nil)
	if len(errs) != 0 {
		t.Fatalf("errs=%v", errs)
	}
	// base normalized=1, additive +2, corrective -0.5 => 2.5
	if out[0].X != 2.5 {
		t.Fatalf("layered X = %v want 2.5", out[0].X)
	}
}

func TestLayer_BaseNormalizedButAdditiveNot(t *testing.T) {
	neutral := mkNeutral(1)
	b1 := mkTarget("b1", KindBase, 0, 1, Vec3{X: 1})
	b2 := mkTarget("b2", KindBase, 0, 1, Vec3{X: 1})
	a1 := mkTarget("a1", KindAdditive, 0, 1, Vec3{X: 10})
	m := NewLayerMixer(neutral)

	// base weights 3,3 normalize to 0.5,0.5 => base contributes 1
	// additive 0.2 stays 0.2 => +2
	out, _, _ := m.Evaluate([]*Target{b1, b2, a1},
		map[string]float64{"b1": 3, "b2": 3, "a1": 0.2}, DefaultWeightPolicy(), nil)
	if d := out[0].X - 3.0; d > 1e-12 || d < -1e-12 {
		t.Fatalf("X = %v want 3", out[0].X)
	}
}

func TestLayer_PriorityThenNameOrder(t *testing.T) {
	// Within corrective layer, summation order is (Priority, Name).
	neutral := mkNeutral(1)
	lo := mkTarget("lo", KindCorrective, 0, 1, Vec3{X: 1})
	hi := mkTarget("hi", KindCorrective, 1, 1, Vec3{X: 1})
	m := NewLayerMixer(neutral)
	p := WeightPolicy{Min: 0, Max: 1, Clamp: true, Normalize: false, SkipInvalid: true}

	o1, _, _ := m.Evaluate([]*Target{hi, lo}, map[string]float64{"lo": 1, "hi": 1}, p, nil)
	o2, _, _ := m.Evaluate([]*Target{lo, hi}, map[string]float64{"lo": 1, "hi": 1}, p, nil)
	if o1[0] != o2[0] {
		t.Fatalf("priority order ambiguous: %v vs %v", o1[0], o2[0])
	}
}
