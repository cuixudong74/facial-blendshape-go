package blendshape

import "sort"

// LayeredBlend 按确定顺序组合三层形变：
//
//  1. 基础层：权重按 policy 归一化后加权混合到基础网格；
//  2. 叠加层：在基础层结果上按权重直接叠加（不归一化，钳制到 [0,1]）；
//  3. 校正层：最后应用；有效权重 = 自身权重 × 各驱动目标（基础层归一化后）权重乘积。
//
// 层间顺序固定（基础 → 叠加 → 校正），层内按目标名排序遍历，
// 结果与输入组合顺序无关、可复现。损坏目标按层跳过并记录。
func (b *Blender) LayeredBlend(out []float32, weights map[string]float64, targets map[string]*MorphTarget, policy NormalizePolicy, issues *IssueList) {
	baseW := map[string]float64{}
	addW := map[string]float64{}
	corrW := map[string]float64{}

	// 分层归属只依赖目标定义与权重值，与遍历顺序无关。
	for name, w := range weights {
		t, ok := targets[name]
		if !ok {
			*issues = append(*issues, Issue{Target: name, Reason: "target not found"})
			continue
		}
		switch t.Layer {
		case LayerBase:
			baseW[name] = w
		case LayerAdditive:
			addW[name] = clamp01(w)
		case LayerCorrective:
			corrW[name] = clamp01(w)
		}
	}

	// 第 1 层：基础层，归一化后混合（含基础网格拷贝）。
	NormalizeWeights(baseW, policy)
	b.Blend(out, baseW, targets, issues)

	// 第 2 层：叠加层，直接累加。
	b.addWeighted(out, addW, targets, issues)

	// 第 3 层：校正层，按驱动目标乘积计算有效权重。
	if len(corrW) > 0 {
		eff := make(map[string]float64, len(corrW))
		names := make([]string, 0, len(corrW))
		for name := range corrW {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			w := corrW[name]
			t := targets[name]
			for _, drv := range t.Drivers {
				w *= baseW[drv] // 驱动目标缺席时权重为 0，校正不生效
			}
			if w != 0 {
				eff[name] = w
			}
		}
		b.addWeighted(out, eff, targets, issues)
	}
}

func clamp01(w float64) float64 {
	if w != w { // NaN 视为非法，置 0
		return 0
	}
	if w < 0 {
		return 0
	}
	if w > 1 {
		return 1
	}
	return w
}
