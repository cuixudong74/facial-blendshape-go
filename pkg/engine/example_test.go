package engine_test

import (
	"fmt"

	"github.com/cuixudong74/facial-blendshape-go/pkg/anim"
	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
	"github.com/cuixudong74/facial-blendshape-go/pkg/engine"
	"github.com/cuixudong74/facial-blendshape-go/pkg/store"
)

func Example() {
	const V = 2
	neutral := []blendshape.Vec3{{}, {}}
	st := store.NewStore()
	smile := make([]blendshape.Vec3, V)
	smile[0] = blendshape.Vec3{Y: 1}
	st.Publish([]*blendshape.Target{
		{Name: "smile", Kind: blendshape.KindBase, VertexCount: V, Deltas: smile},
	}, V)

	eng := engine.NewEngine(neutral, blendshape.DefaultWeightPolicy(), st)
	clip := &anim.Clip{Name: "smile-in", Duration: 1, Tracks: []*anim.Track{
		{Target: "smile", Keys: []anim.Keyframe{{Time: 0, Value: 0}, {Time: 1, Value: 1}}},
	}}
	ch := anim.NewChannel(clip, 0, 1, anim.DefaultSampleOptions())

	f := eng.EvaluateAt([]*anim.Channel{ch}, 0.5, nil)
	// Base layer normalizes a single active target to 1, so the posed vertex
	// moves the full delta even at clip time 0.5 (raw track value 0.5).
	fmt.Printf("v%d Y0=%.2f issues=%d errors=%d\n", f.Version, f.Vertices[0].Y, len(f.Issues), len(f.Errors))
	// Output: v1 Y0=1.00 issues=0 errors=0
}
