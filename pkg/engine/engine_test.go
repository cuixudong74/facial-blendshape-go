package engine

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/cuixudong74/facial-blendshape-go/pkg/anim"
	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
	"github.com/cuixudong74/facial-blendshape-go/pkg/store"
)

const V = 128

func neutral() []blendshape.Vec3 {
	v := make([]blendshape.Vec3, V)
	for i := range v {
		v[i] = blendshape.Vec3{X: float64(i)}
	}
	return v
}

func tgt(name string, kind blendshape.TargetKind, prio int, x float64) *blendshape.Target {
	d := make([]blendshape.Vec3, V)
	for i := range d {
		d[i] = blendshape.Vec3{X: x}
	}
	return &blendshape.Target{Name: name, Kind: kind, Priority: prio, VertexCount: V, Deltas: d}
}

func TestEngine_WeightsAndVersion(t *testing.T) {
	st := store.NewStore()
	st.Publish([]*blendshape.Target{tgt("smile", blendshape.KindBase, 0, 1)}, V)
	e := NewEngine(neutral(), blendshape.DefaultWeightPolicy(), st)

	f := e.EvaluateWeights(map[string]float64{"smile": 1}, nil)
	if f.Version != 1 || f.Vertices[0].X != 1 {
		t.Fatalf("frame ver=%d x=%v", f.Version, f.Vertices[0].X)
	}
	// neutral preserved at other vertices? delta applies all, X = i+1
	if f.Vertices[10].X != 11 {
		t.Fatalf("pose wrong: %v", f.Vertices[10])
	}
}

func TestEngine_MultiChannelMerge(t *testing.T) {
	st := store.NewStore()
	st.Publish([]*blendshape.Target{
		tgt("a", blendshape.KindBase, 0, 1),
		tgt("b", blendshape.KindBase, 0, 1),
	}, V)
	e := NewEngine(neutral(), blendshape.DefaultWeightPolicy(), st)

	clipA := &anim.Clip{Name: "ca", Duration: 1, Tracks: []*anim.Track{
		{Target: "a", Keys: []anim.Keyframe{{Time: 0, Value: 1}}},
	}}
	clipB := &anim.Clip{Name: "cb", Duration: 1, Tracks: []*anim.Track{
		{Target: "b", Keys: []anim.Keyframe{{Time: 0, Value: 1}}},
	}}
	chans := []*anim.Channel{
		anim.NewChannel(clipA, 0, 1, anim.DefaultSampleOptions()),
		anim.NewChannel(clipB, 0, 1, anim.DefaultSampleOptions()),
	}
	f := e.EvaluateAt(chans, 0, nil)
	// base normalized 0.5/0.5 => total delta 1 on every vertex
	if f.Vertices[0].X != 1 {
		t.Fatalf("merged pose X = %v want 1", f.Vertices[0].X)
	}
}

func TestEngine_ConcurrentConsistency(t *testing.T) {
	st := store.NewStore()
	st.Publish([]*blendshape.Target{
		tgt("a", blendshape.KindBase, 0, 1),
		tgt("b", blendshape.KindAdditive, 0, 2),
		tgt("c", blendshape.KindCorrective, 0, -0.5),
	}, V)
	e := NewEngine(neutral(), blendshape.DefaultWeightPolicy(), st)
	w := map[string]float64{"a": 1, "b": 0.5, "c": 1}

	ref := e.EvaluateWeights(w, make([]blendshape.Vec3, V))

	const goroutines = 16
	var wg sync.WaitGroup
	errCh := make(chan string, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out := make([]blendshape.Vec3, V)
			for i := 0; i < 2000; i++ {
				f := e.EvaluateWeights(w, out)
				for v := 0; v < V; v++ {
					if f.Vertices[v] != ref.Vertices[v] {
						errCh <- "divergent vertex"
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	select {
	case msg := <-errCh:
		t.Fatal(msg)
	default:
	}
}

func TestEngine_FrameNeverTornDuringHotReplace(t *testing.T) {
	st := store.NewStore()
	st.Publish([]*blendshape.Target{
		tgt("a", blendshape.KindBase, 0, 1),
		tgt("b", blendshape.KindBase, 0, 1),
		tgt("c", blendshape.KindCorrective, 0, 0.25),
	}, V)
	e := NewEngine(neutral(), blendshape.DefaultWeightPolicy(), st)
	// weight on c too: if a frame mixed a 3-target set with a 2-target weight
	// map, c would silently vanish; version/result mismatch is what we hunt.
	w := map[string]float64{"a": 1, "b": 1, "c": 1}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	bad := make(chan string, 16)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			if i%2 == 0 {
				// same target names as v2, only c payload differs (true replace)
				st.Publish([]*blendshape.Target{
					tgt("a", blendshape.KindBase, 0, 1),
					tgt("b", blendshape.KindBase, 0, 1),
					tgt("c", blendshape.KindCorrective, 0, 0),
				}, V)
			} else {
				st.Publish([]*blendshape.Target{
					tgt("a", blendshape.KindBase, 0, 1),
					tgt("b", blendshape.KindBase, 0, 1),
					tgt("c", blendshape.KindCorrective, 0, 0.25),
				}, V)
			}
			runtime.Gosched()
		}
	}()

	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out := make([]blendshape.Vec3, V)
			ref2 := NewEngine(neutral(), blendshape.DefaultWeightPolicy(), store2(V))
			ref3 := NewEngine(neutral(), blendshape.DefaultWeightPolicy(), store3(V))
			exp2 := ref2.EvaluateWeights(w, nil).Vertices
			exp3 := ref3.EvaluateWeights(w, nil).Vertices
			for i := 0; i < 1500; i++ {
				f := e.EvaluateWeights(w, out)
				// Independently recompute the frame shape that f.Version must
				// have; exact agreement proves the frame evaluated against one
				// fully published set.
				var expected []blendshape.Vec3
				if f.Version >= 2 && f.Version%2 == 0 { // v2,v4,... c=0
					expected = exp2
				} else { // v1 or v3,v5,... c=0.25
					expected = exp3
				}
				for v := range f.Vertices {
					if f.Vertices[v] != expected[v] {
						bad <- "frame did not match its declared snapshot version"
						return
					}
				}
			}
		}()
	}
	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()
	select {
	case msg := <-bad:
		t.Fatal(msg)
	default:
	}
}

// store2/store3 build deterministic stores representing the two set shapes, so
// tests can recompute what a frame of a given version must look like.
func store2(v int) *store.Store {
	s := store.NewStore()
	s.Publish([]*blendshape.Target{
		tgt("a", blendshape.KindBase, 0, 1),
		tgt("b", blendshape.KindBase, 0, 1),
		tgt("c", blendshape.KindCorrective, 0, 0),
	}, v)
	return s
}

func store3(v int) *store.Store {
	s := store.NewStore()
	s.Publish([]*blendshape.Target{
		tgt("a", blendshape.KindBase, 0, 1),
		tgt("b", blendshape.KindBase, 0, 1),
		tgt("c", blendshape.KindCorrective, 0, 0.25),
	}, v)
	return s
}
