package blendshape

import (
	"math"
	"testing"
)

func TestBlendWeightedSum(t *testing.T) {
	mesh := makeMesh(4)
	b := NewBlender(mesh)
	targets := map[string]*MorphTarget{
		"smile": makeTarget("smile", LayerBase, 4, 1, 0, 0),
		"blink": makeTarget("blink", LayerBase, 4, 0, 2, 0),
	}
	out := make([]float32, 12)
	var issues IssueList
	b.Blend(out, map[string]float64{"smile": 0.5, "blink": 0.25}, targets, &issues)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	// 顶点 2 基准 (2,2,2)：x += 0.5*1, y += 0.25*2
	if !close32(out[6], 2.5) || !close32(out[7], 2.5) || !close32(out[8], 2.0) {
		t.Fatalf("vertex 2 = %v, want (2.5,2.5,2.0)", out[6:9])
	}
	// 基础网格不被修改
	if mesh.Positions[6] != 2 {
		t.Fatalf("base mesh mutated")
	}
}

func TestBlendSkipsMissingAndCorrupt(t *testing.T) {
	mesh := makeMesh(2)
	b := NewBlender(mesh)
	bad := makeTarget("bad", LayerBase, 2, 1, 1, 1)
	bad.Deltas = bad.Deltas[:4] // 维度损坏
	nan := makeTarget("nan", LayerBase, 2, 0, 0, 0)
	nan.Deltas[0] = float32(math.NaN())
	targets := map[string]*MorphTarget{"bad": bad, "nan": nan}
	out := make([]float32, 6)
	var issues IssueList
	b.Blend(out, map[string]float64{"bad": 1, "nan": 1, "ghost": 1}, targets, &issues)
	if len(issues) != 3 {
		t.Fatalf("want 3 issues, got %v", issues)
	}
	seen := map[string]bool{}
	for _, i := range issues {
		seen[i.Target] = true
		if i.Reason == "" {
			t.Fatalf("empty reason for %q", i.Target)
		}
	}
	for _, want := range []string{"bad", "nan", "ghost"} {
		if !seen[want] {
			t.Fatalf("missing issue for %q", want)
		}
	}
	// 整帧不崩溃，输出保持基础网格
	for i, v := range out {
		if v != mesh.Positions[i] {
			t.Fatalf("out[%d] = %v, want base %v", i, v, mesh.Positions[i])
		}
	}
}

func TestBlendReproducible(t *testing.T) {
	mesh := makeMesh(8)
	b := NewBlender(mesh)
	targets := map[string]*MorphTarget{
		"t1": makeTarget("t1", LayerBase, 8, 0.1, 0.2, 0.3),
		"t2": makeTarget("t2", LayerBase, 8, -0.4, 0.5, 0.6),
		"t3": makeTarget("t3", LayerBase, 8, 0.7, -0.8, 0.9),
	}
	w := map[string]float64{"t1": 0.3, "t2": 0.5, "t3": 0.2}
	out1 := make([]float32, 24)
	out2 := make([]float32, 24)
	var i1, i2 IssueList
	b.Blend(out1, w, targets, &i1)
	b.Blend(out2, w, targets, &i2)
	for i := range out1 {
		if out1[i] != out2[i] {
			t.Fatalf("run mismatch at %d: %v vs %v", i, out1[i], out2[i])
		}
	}
}
