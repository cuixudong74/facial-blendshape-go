// Command demo drives a tiny facial rig, then hot-replaces a target mid-run to
// show that animation keeps producing complete, correctly-versioned frames.
package main

import (
	"fmt"
	"time"

	"github.com/cuixudong74/facial-blendshape-go/pkg/anim"
	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
	"github.com/cuixudong74/facial-blendshape-go/pkg/engine"
	"github.com/cuixudong74/facial-blendshape-go/pkg/store"
)

const V = 3

func target(name string, kind blendshape.TargetKind, y float64) *blendshape.Target {
	d := make([]blendshape.Vec3, V)
	for i := range d {
		d[i] = blendshape.Vec3{Y: y}
	}
	return &blendshape.Target{Name: name, Kind: kind, VertexCount: V, Deltas: d}
}

func main() {
	neutral := make([]blendshape.Vec3, V)
	st := store.NewStore()
	st.Publish([]*blendshape.Target{
		target("smile", blendshape.KindBase, 1),
		target("jaw", blendshape.KindAdditive, 0),
		target("lip_fix", blendshape.KindCorrective, 0),
	}, V)

	eng := engine.NewEngine(neutral, blendshape.DefaultWeightPolicy(), st)

	clip := &anim.Clip{Name: "smile-jaw", Duration: 2, Tracks: []*anim.Track{
		{Target: "smile", Keys: []anim.Keyframe{{Time: 0, Value: 0}, {Time: 1, Value: 1}, {Time: 2, Value: 0}}},
		{Target: "jaw", Keys: []anim.Keyframe{{Time: 0, Value: 0}, {Time: 1, Value: 0.5}, {Time: 2, Value: 0}}},
	}}
	ch := anim.NewChannel(clip, 0, 1, anim.DefaultSampleOptions())

	// Simulate a hot replacement landing halfway through playback.
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, errs := st.Replace([]*blendshape.Target{
			target("smile", blendshape.KindBase, 2), // stronger smile
			target("lip_fix", blendshape.KindCorrective, -0.25),
		}, nil, V)
		if len(errs) > 0 {
			fmt.Println("replace rejected:", errs)
		} else {
			fmt.Println(">> hot replaced: smile strength doubled, corrective added")
		}
	}()

	start := time.Now()
	out := make([]blendshape.Vec3, V)
	for frame := 0; frame < 6; frame++ {
		t := float64(frame) * 0.4 // 0.4s per frame; continuous time, fixed sample rate is irrelevant
		f := eng.EvaluateAt([]*anim.Channel{ch}, t, out)
		fmt.Printf("t=%.1fs v%d  Y=[%.3f %.3f %.3f] issues=%d errs=%d\n",
			t, f.Version, f.Vertices[0].Y, f.Vertices[1].Y, f.Vertices[2].Y,
			len(f.Issues), len(f.Errors))
		time.Sleep(40 * time.Millisecond)
	}
	fmt.Printf("elapsed wall time: %v\n", time.Since(start).Round(time.Millisecond))
}
