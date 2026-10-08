package blendshape

import (
	"math"
	"sort"
)

// NormalizePolicy 定义权重归一化策略。
type NormalizePolicy struct {
	// ClampNegative 为 true 时负权重钳制到 0；为 false 时负权重视为非法置 0。
	// 两者效果一致，保留开关以便未来扩展（如允许负向形变）。
	ClampNegative bool
	// MaxSum 为权重和上限；超过时按比例整体缩放，保证结果可复现。
	// <= 0 表示不限制权重和。
	MaxSum float64
}

// DefaultNormalizePolicy 为默认策略：负值钳零、NaN/Inf 置零、权重和上限 1。
var DefaultNormalizePolicy = NormalizePolicy{ClampNegative: true, MaxSum: 1.0}

// NormalizeWeights 按策略原地归一化权重：
//  1. NaN / ±Inf 置 0；
//  2. 负权重钳制到 0；
//  3. 权重和超过 MaxSum 时按比例缩放。
//
// 返回被修正过的键列表（升序，确定顺序）。给定输入下输出可复现。
func NormalizeWeights(weights map[string]float64, p NormalizePolicy) []string {
	var fixed []string
	// 先按固定顺序（键升序）做合法性修正，再按同一顺序求和，
	// 保证浮点累加顺序确定、结果可复现。
	keys := make([]string, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sum := 0.0
	for _, k := range keys {
		w := weights[k]
		switch {
		case math.IsNaN(w) || math.IsInf(w, 0):
			weights[k] = 0
			fixed = append(fixed, k)
		case w < 0:
			weights[k] = 0
			fixed = append(fixed, k)
		default:
			sum += w
		}
	}
	if p.MaxSum > 0 && sum > p.MaxSum {
		scale := p.MaxSum / sum
		for _, k := range keys {
			if weights[k] != 0 {
				weights[k] *= scale
				fixed = append(fixed, k)
			}
		}
	}
	if len(fixed) > 0 {
		sort.Strings(fixed)
	}
	return fixed
}
