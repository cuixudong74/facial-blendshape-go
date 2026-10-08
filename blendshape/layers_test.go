package blendshape

import "testing"

// 基础层归一化 → 叠加层直接叠加 → 校正层按驱动乘积最后应用。
func TestLayeredBlendOrder(t *testing.T) {
	mesh := makeMesh(1) // 顶点 (0,0,0)
	b := NewBlender(mesh)
	corr := makeTarget("corr", LayerCorrective, 1, 0, 0, 10)
	corr.Drivers = []string{"jaw", "smile"}
	targets := map[string]*MorphTarget{
		"jaw":     makeTarget("jaw", LayerBase, 1, 1, 0, 0),
		"smile":   makeTarget("smile", LayerBase, 1, 0, 2, 0),
		"wrinkle": makeTarget("wrinkle", LayerAdditive, 1, 0, 0, 1),
		"corr":    corr,
	}
	// 基础层权重和 2.0 → 归一化为 jaw=0.25, smile=0.75
	w := map[string]float64{"jaw": 0.5, "smile": 1.5, "wrinkle": 0.5, "corr": 0.8}
	out := make([]float32, 3)
	var issues IssueList
	b.LayeredBlend(out, w, targets, DefaultNormalizePolicy, &issues)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %v", issues)
	}
	// x = 0.25*1 = 0.25
	// y = 0.75*2 = 1.5
	// z = 0.5*1 (叠加) + 0.8*0.25*0.75*10 (校正) = 0.5 + 1.5 = 2.0
	if !close32(out[0], 0.25) || !close32(out[1], 1.5) || !close32(out[2], 2.0) {
		t.Fatalf("out = %v, want (0.25, 1.5, 2.0)", out)
	}
}

// 层内结果与 map 遍历顺序无关：多次求值结果一致。
func TestLayeredBlendOrderIndependent(t *testing.T) {
	mesh := makeMesh(4)
	b := NewBlender(mesh)
	targets := map[string]*MorphTarget{
		"a": makeTarget("a", LayerBase, 4, 0.1, 0, 0),
		"b": makeTarget("b", LayerBase, 4, 0, 0.2, 0),
		"c": makeTarget("c", LayerAdditive, 4, 0, 0, 0.3),
		"d": makeTarget("d", LayerCorrective, 4, 0.4, 0, 0),
	}
	w := map[string]float64{"a": 0.3, "b": 0.7, "c": 0.5, "d": 0.9}
	ref := make([]float32, 12)
	var iss IssueList
	b.LayeredBlend(ref, w, targets, DefaultNormalizePolicy, &iss)
	for trial := 0; trial < 20; trial++ {
		out := make([]float32, 12)
		var iss2 IssueList
		b.LayeredBlend(out, w, targets, DefaultNormalizePolicy, &iss2)
		for i := range ref {
			if out[i] != ref[i] {
				t.Fatalf("trial %d: out[%d] = %v, want %v", trial, i, out[i], ref[i])
			}
		}
	}
}

// 校正目标的驱动目标缺席时校正不生效。
func TestCorrectiveWithoutDriverIsNoop(t *testing.T) {
	mesh := makeMesh(1)
	b := NewBlender(mesh)
	corr := makeTarget("corr", LayerCorrective, 1, 5, 5, 5)
	corr.Drivers = []string{"jaw"}
	targets := map[string]*MorphTarget{
		"jaw":  makeTarget("jaw", LayerBase, 1, 1, 0, 0),
		"corr": corr,
	}
	out := make([]float32, 3)
	var issues IssueList
	b.LayeredBlend(out, map[string]float64{"corr": 1.0}, targets, DefaultNormalizePolicy, &issues)
	if out[0] != 0 || out[1] != 0 || out[2] != 0 {
		t.Fatalf("corrective applied without driver: %v", out)
	}
}

// 损坏目标在分层评估中被跳过且不中断整帧。
func TestLayeredBlendSkipsCorrupt(t *testing.T) {
	mesh := makeMesh(2)
	b := NewBlender(mesh)
	bad := makeTarget("bad", LayerAdditive, 2, 9, 9, 9)
	bad.VertexCount = 99 // 维度不一致
	targets := map[string]*MorphTarget{
		"ok":  makeTarget("ok", LayerBase, 2, 1, 0, 0),
		"bad": bad,
	}
	out := make([]float32, 6)
	var issues IssueList
	b.LayeredBlend(out, map[string]float64{"ok": 1, "bad": 1}, targets, DefaultNormalizePolicy, &issues)
	if len(issues) != 1 || issues[0].Target != "bad" {
		t.Fatalf("want 1 issue for bad, got %v", issues)
	}
	if !close32(out[0], 1) { // ok 目标仍生效
		t.Fatalf("out[0] = %v, want 1", out[0])
	}
}
