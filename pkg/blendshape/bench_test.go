package blendshape

import "testing"

// benchMix scales target count T and vertex count V. Hot-loop cost is O(V*T)
// and the reused output buffer means the steady-state frame allocates only
// small bookkeeping (group slices/maps), never the mesh.
func benchMix(b *testing.B, tcount, vcount int) {
	neutral := make([]Vec3, vcount)
	targets := make([]*Target, tcount)
	weights := map[string]float64{}
	for i := range targets {
		name := "t" + itoaBench(i)
		targets[i] = mkTarget(name, KindBase, i%4, vcount, Vec3{X: 0.01, Y: 0.02, Z: 0.03})
		weights[name] = 1
	}
	m := NewLayerMixer(neutral)
	out := make([]Vec3, vcount)
	p := DefaultWeightPolicy()
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, _ = m.Evaluate(targets, weights, p, out)
	}
}

func BenchmarkMix_1k_verts_8_targets(b *testing.B)   { benchMix(b, 8, 1000) }
func BenchmarkMix_10k_verts_32_targets(b *testing.B) { benchMix(b, 32, 10000) }
func BenchmarkMix_50k_verts_64_targets(b *testing.B) { benchMix(b, 64, 50000) }

func itoaBench(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
