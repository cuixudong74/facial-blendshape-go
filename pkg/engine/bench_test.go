package engine

import (
	"testing"

	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
	"github.com/cuixudong74/facial-blendshape-go/pkg/store"
)

func BenchmarkEngineFrame_10k_verts_32_targets(b *testing.B) {
	const vcount, tcount = 10000, 32
	st := store.NewStore()
	targets := make([]*blendshape.Target, tcount)
	weights := map[string]float64{}
	for i := range targets {
		name := "t" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		kind := blendshape.KindBase
		if i%3 == 1 {
			kind = blendshape.KindAdditive
		}
		if i%3 == 2 {
			kind = blendshape.KindCorrective
		}
		d := make([]blendshape.Vec3, vcount)
		for j := range d {
			d[j] = blendshape.Vec3{X: 0.01, Y: 0.01, Z: 0.01}
		}
		targets[i] = &blendshape.Target{Name: name, Kind: kind, VertexCount: vcount, Deltas: d}
		weights[name] = 0.5
	}
	st.Publish(targets, vcount)
	e := NewEngine(make([]blendshape.Vec3, vcount), blendshape.DefaultWeightPolicy(), st)
	out := make([]blendshape.Vec3, vcount)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		f := e.EvaluateWeights(weights, out)
		_ = f
	}
}
