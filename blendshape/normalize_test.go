package blendshape

import (
	"math"
	"testing"
)

func TestNormalizeClampsNegativeAndNaN(t *testing.T) {
	w := map[string]float64{"a": -0.5, "b": math.NaN(), "c": math.Inf(1), "d": 0.4}
	fixed := NormalizeWeights(w, DefaultNormalizePolicy)
	if w["a"] != 0 || w["b"] != 0 || w["c"] != 0 {
		t.Fatalf("illegal weights not zeroed: %v", w)
	}
	if w["d"] != 0.4 {
		t.Fatalf("legal weight changed: %v", w)
	}
	if len(fixed) != 3 {
		t.Fatalf("want 3 fixed keys, got %v", fixed)
	}
}

func TestNormalizeScalesDownOverflow(t *testing.T) {
	w := map[string]float64{"a": 0.6, "b": 0.6}
	NormalizeWeights(w, DefaultNormalizePolicy)
	sum := w["a"] + w["b"]
	if !close64(sum, 1.0) {
		t.Fatalf("sum = %v, want 1.0", sum)
	}
	if !close64(w["a"], w["b"]) {
		t.Fatalf("scaling not proportional: %v", w)
	}
}

func TestNormalizeDeterministic(t *testing.T) {
	mk := func() map[string]float64 {
		return map[string]float64{"x": 0.9, "y": 0.8, "z": 0.7}
	}
	w1, w2 := mk(), mk()
	NormalizeWeights(w1, DefaultNormalizePolicy)
	NormalizeWeights(w2, DefaultNormalizePolicy)
	for k := range w1 {
		if w1[k] != w2[k] {
			t.Fatalf("non-deterministic result for %q: %v vs %v", k, w1[k], w2[k])
		}
	}
}

func TestNormalizeNoLimitWhenMaxSumZero(t *testing.T) {
	w := map[string]float64{"a": 2.0, "b": 3.0}
	NormalizeWeights(w, NormalizePolicy{MaxSum: 0})
	if w["a"] != 2.0 || w["b"] != 3.0 {
		t.Fatalf("weights scaled despite MaxSum=0: %v", w)
	}
}
