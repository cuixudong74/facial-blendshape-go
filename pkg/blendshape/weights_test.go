package blendshape

import (
	"math"
	"testing"
)

func TestNormalizeWeights_SumToOne(t *testing.T) {
	p := WeightPolicy{Min: 0, Max: 1, Clamp: true, Normalize: true, SkipInvalid: true}
	out, issues := NormalizeWeights(map[string]float64{"a": 0.6, "b": 0.4}, p)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	var sum float64
	for _, v := range out {
		sum += v
	}
	if math.Abs(sum-1) > 1e-12 {
		t.Fatalf("weights sum = %v, want 1", sum)
	}
	if math.Abs(out["a"]-0.6) > 1e-12 || math.Abs(out["b"]-0.4) > 1e-12 {
		t.Fatalf("normalized = %v", out)
	}

	// Oversized raw weights clamp then renormalize to equal shares.
	out2, issues2 := NormalizeWeights(map[string]float64{"a": 2, "b": 2}, p)
	if len(issues2) != 2 {
		t.Fatalf("want 2 clamp issues, got %v", issues2)
	}
	if math.Abs(out2["a"]-0.5) > 1e-12 || math.Abs(out2["b"]-0.5) > 1e-12 {
		t.Fatalf("normalized clamped = %v", out2)
	}
}

func TestNormalizeWeights_Clamp(t *testing.T) {
	p := DefaultWeightPolicy()
	out, issues := NormalizeWeights(map[string]float64{"a": 3, "b": -2}, p)
	if len(issues) != 2 {
		t.Fatalf("want 2 clamp issues, got %d: %v", len(issues), issues)
	}
	// clamped to 1 and 0 => after normalization a = 1
	if out["a"] != 1 {
		t.Fatalf("a = %v, want 1", out["a"])
	}
	if _, ok := out["b"]; !ok {
		t.Fatalf("b should remain present as 0? got %v", out)
	}
}

func TestNormalizeWeights_NaNDropped(t *testing.T) {
	p := DefaultWeightPolicy() // SkipInvalid true
	out, issues := NormalizeWeights(map[string]float64{"a": math.NaN(), "b": 1}, p)
	if len(issues) != 1 || issues[0].Target != "a" {
		t.Fatalf("want issue on a, got %v", issues)
	}
	if out["a"] != 0 || out["b"] != 1 {
		t.Fatalf("unexpected out %v", out)
	}
}

func TestNormalizeWeights_NoClampRejects(t *testing.T) {
	p := WeightPolicy{Min: 0, Max: 1, Clamp: false, Normalize: true, SkipInvalid: true}
	out, issues := NormalizeWeights(map[string]float64{"a": 5, "b": 1}, p)
	if len(issues) != 1 {
		t.Fatalf("want 1 rejection, got %v", issues)
	}
	if _, ok := out["a"]; ok {
		t.Fatalf("a should be dropped: %v", out)
	}
	if out["b"] != 1 {
		t.Fatalf("b = %v want 1", out["b"])
	}
}

func TestNormalizeWeights_Reproducible(t *testing.T) {
	p := DefaultWeightPolicy()
	in := map[string]float64{"z": 0.7, "a": 1.3, "m": 2.1, "b": 0.4}
	r1, _ := NormalizeWeights(in, p)
	r2, _ := NormalizeWeights(in, p)
	for k := range in {
		if r1[k] != r2[k] {
			t.Fatalf("non-reproducible weight for %s: %v vs %v", k, r1[k], r2[k])
		}
	}
}

func TestNormalizeWeights_EmptyAndAllZero(t *testing.T) {
	p := DefaultWeightPolicy()
	out, issues := NormalizeWeights(map[string]float64{}, p)
	if len(out) != 0 || len(issues) != 0 {
		t.Fatalf("empty input: %v %v", out, issues)
	}
	out2, _ := NormalizeWeights(map[string]float64{"a": 0, "b": 0}, p)
	if out2["a"] != 0 || out2["b"] != 0 {
		t.Fatalf("all-zero must not divide/explode: %v", out2)
	}
}

func TestNormalizeWeights_NaNAsZeroWhenNotSkipping(t *testing.T) {
	p := WeightPolicy{Min: 0, Max: 1, Clamp: true, Normalize: false, SkipInvalid: false}
	out, issues := NormalizeWeights(map[string]float64{"a": math.NaN(), "b": 0.5}, p)
	if len(issues) != 1 || out["a"] != 0 || out["b"] != 0.5 {
		t.Fatalf("NaN-as-zero policy wrong: %v %v", out, issues)
	}
}
