package blendshape

import (
	"fmt"
	"testing"
)

// 基准：顶点规模 × 活跃目标数。混合成本应为 O(活跃目标数 × 顶点数)，
// 与注册表中目标总数无关；输出缓冲复用，每帧零整份网格拷贝（除基准网格本身一次 copy）。
func BenchmarkLayeredBlend(b *testing.B) {
	for _, verts := range []int{1000, 10000, 50000} {
		for _, active := range []int{4, 32, 128} {
			name := fmt.Sprintf("verts=%d/active=%d", verts, active)
			b.Run(name, func(b *testing.B) {
				mesh := makeMesh(verts)
				bl := NewBlender(mesh)
				targets := map[string]*MorphTarget{}
				w := map[string]float64{}
				// 注册 4 倍于活跃数的目标，验证成本与总数无关。
				for i := 0; i < 4*active; i++ {
					n := fmt.Sprintf("t%03d", i)
					targets[n] = makeTarget(n, LayerBase, verts, 0.001, 0.002, 0.003)
					if i < active {
						w[n] = 0.5
					}
				}
				out := make([]float32, 3*verts)
				var iss IssueList
				b.ReportAllocs()
				b.SetBytes(int64(3 * verts * 4)) // 每帧写出的顶点字节数
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					bl.LayeredBlend(out, w, targets, DefaultNormalizePolicy, &iss)
				}
			})
		}
	}
}

func BenchmarkEngineEvaluate(b *testing.B) {
	mesh := makeMesh(10000)
	e := NewEngine(mesh)
	for i := 0; i < 64; i++ {
		n := fmt.Sprintf("t%02d", i)
		_ = e.Registry().Load(makeTarget(n, LayerBase, 10000, 0.001, 0, 0))
	}
	tracks := make([]Track, 16)
	for i := range tracks {
		tracks[i] = Track{
			Target: fmt.Sprintf("t%02d", i),
			Keys:   []Keyframe{{0, 0}, {0.5, 1}, {1, 0}},
			Interp: InterpLinear,
		}
	}
	e.AddChannel(NewChannel(&Animation{Name: "a", Duration: 1, Tracks: tracks}, LoopRepeat))
	out := make([]float32, 30000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Advance(1.0 / 60.0)
		_ = e.Evaluate(out)
	}
}
