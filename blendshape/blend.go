package blendshape

import (
	"fmt"
	"sort"
)

// Blender 对一组形变目标按权重做加权混合。
// 输出写入调用方提供的缓冲，避免每帧整份网格拷贝；
// 每帧成本为 O(Σ 活跃目标非零增量)，而非 O(目标数 × 顶点数)。
type Blender struct {
	mesh *Mesh
}

// NewBlender 创建基于给定基础网格的混合器。mesh 在运行期只读。
func NewBlender(mesh *Mesh) *Blender {
	return &Blender{mesh: mesh}
}

// VertexCount 返回基础网格顶点数。
func (b *Blender) VertexCount() int { return b.mesh.VertexCount }

// Mesh 返回基础网格（只读）。
func (b *Blender) Mesh() *Mesh { return b.mesh }

// Blend 计算 out = base + Σ w_i * delta_i。
// out 长度必须为 3*VertexCount，由调用方持有，可跨帧复用。
// 维度不一致或数据损坏（含 NaN）的目标被跳过并记录到 issues，不中断整帧。
// 遍历按目标名排序，浮点累加顺序确定，结果可复现。
func (b *Blender) Blend(out []float32, weights map[string]float64, targets map[string]*MorphTarget, issues *IssueList) {
	need := len(b.mesh.Positions)
	if len(out) != need {
		panic(fmt.Sprintf("blendshape: out length %d, want %d", len(out), need))
	}
	copy(out, b.mesh.Positions)
	b.addWeighted(out, weights, targets, issues)
}

// addWeighted 将 Σ w_i * delta_i 累加进 out（不重置 out）。
// 遍历按目标名排序，浮点累加顺序确定，结果可复现。
func (b *Blender) addWeighted(out []float32, weights map[string]float64, targets map[string]*MorphTarget, issues *IssueList) {
	keys := make([]string, 0, len(weights))
	for k, w := range weights {
		if w != 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, name := range keys {
		t, ok := targets[name]
		if !ok {
			*issues = append(*issues, Issue{Target: name, Reason: "target not found"})
			continue
		}
		if err := validateTarget(t, b.mesh.VertexCount); err != nil {
			*issues = append(*issues, Issue{Target: name, Reason: err.Error()})
			continue
		}
		w := float32(weights[name])
		d := t.Deltas
		for i := 0; i < len(d); i++ {
			out[i] += w * d[i]
		}
	}
}

// validateTarget 校验目标数据完整性：维度匹配且不含 NaN/Inf。
func validateTarget(t *MorphTarget, vertexCount int) error {
	if t == nil {
		return fmt.Errorf("nil target")
	}
	if t.VertexCount != vertexCount || len(t.Deltas) != 3*vertexCount {
		return fmt.Errorf("dimension mismatch: got %d verts/%d floats, want %d verts/%d floats",
			t.VertexCount, len(t.Deltas), vertexCount, 3*vertexCount)
	}
	for _, v := range t.Deltas {
		if v != v || v > 1e30 || v < -1e30 { // NaN 或 Inf/溢出
			return fmt.Errorf("corrupted data: NaN/Inf in deltas")
		}
	}
	return nil
}
