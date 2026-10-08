package store

import (
	"sync"
	"testing"

	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
)

func targetAt(name string, n int, x float64) *blendshape.Target {
	d := make([]blendshape.Vec3, n)
	for i := range d {
		d[i] = blendshape.Vec3{X: x}
	}
	return &blendshape.Target{Name: name, Kind: blendshape.KindBase, VertexCount: n, Deltas: d}
}

func TestPublish_AtomicVersioning(t *testing.T) {
	st := NewStore()
	s1, errs := st.Publish([]*blendshape.Target{targetAt("a", 2, 1)}, 2)
	if len(errs) != 0 {
		t.Fatalf("publish: %v", errs)
	}
	if s1.Version() != 1 {
		t.Fatalf("version = %d want 1", s1.Version())
	}
	if got := st.Current(); got.Version() != 1 {
		t.Fatalf("current version = %d", got.Version())
	}
	s2, _ := st.Publish([]*blendshape.Target{targetAt("a", 2, 2), targetAt("b", 2, 3)}, 2)
	if s2.Version() != 2 || s2.Size() != 2 {
		t.Fatalf("v2 = version %d size %d", s2.Version(), s2.Size())
	}
}

func TestPublish_BadBatchRejectedWhole(t *testing.T) {
	st := NewStore()
	s1, _ := st.Publish([]*blendshape.Target{targetAt("a", 2, 1)}, 2)

	// one good + one dimension-mismatched target: whole batch rejected.
	bad := &blendshape.Target{Name: "bad", Kind: blendshape.KindBase, VertexCount: 9,
		Deltas: make([]blendshape.Vec3, 9)}
	snap, errs := st.Publish([]*blendshape.Target{targetAt("a", 2, 5), bad}, 2)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d", len(errs))
	}
	if te, ok := errs[0].(*blendshape.TargetError); !ok || te.Target != "bad" {
		t.Fatalf("error must name bad target: %v", errs[0])
	}
	if snap.Version() != s1.Version() {
		t.Fatal("snapshot must stay at old version after failed publish")
	}
	tg, _ := st.Current().Get("a")
	if tg.Deltas[0].X != 1 {
		t.Fatal("old target content changed after failed publish")
	}
}

func TestReplace_UpsertAndRemove(t *testing.T) {
	st := NewStore()
	st.Publish([]*blendshape.Target{targetAt("a", 2, 1), targetAt("b", 2, 1)}, 2)
	snap, errs := st.Replace([]*blendshape.Target{targetAt("a", 2, 9)}, []string{"b"}, 2)
	if len(errs) != 0 {
		t.Fatalf("replace: %v", errs)
	}
	if snap.Size() != 1 {
		t.Fatalf("size=%d want 1", snap.Size())
	}
	tg, ok := snap.Get("a")
	if !ok || tg.Deltas[0].X != 9 {
		t.Fatal("upsert did not take effect")
	}
	if _, ok := snap.Get("b"); ok {
		t.Fatal("b not removed")
	}
}

func TestReplace_BadUpsertKeepsOld(t *testing.T) {
	st := NewStore()
	v1, _ := st.Publish([]*blendshape.Target{targetAt("a", 2, 1)}, 2)
	bad := &blendshape.Target{Name: "x", Kind: blendshape.KindBase, VertexCount: 1}
	snap, errs := st.Replace([]*blendshape.Target{bad}, nil, 2)
	if len(errs) != 1 || snap.Version() != v1.Version() {
		t.Fatal("bad replace must not change version")
	}
}

func TestReaderNeverTorn(t *testing.T) {
	const V = 64
	st := NewStore()
	st.Publish([]*blendshape.Target{targetAt("a", V, 1), targetAt("b", V, 1)}, V)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// readers: every observed version must be internally complete and stable.
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				snap := st.Current()
				var sum float64
				for _, name := range snap.Names() {
					tg, ok := snap.Get(name)
					if !ok {
						t.Errorf("name %s missing in snapshot %d", name, snap.Version())
						return
					}
					if len(tg.Deltas) != V {
						t.Errorf("torn read: %s has %d deltas in v%d", name, len(tg.Deltas), snap.Version())
						return
					}
					sum += tg.Deltas[0].X
				}
				// v1: two targets each X=1; later versions: see writer.
				_ = sum
			}
		}()
	}

	// writer: alternating full sets
	for i := 0; i < 200; i++ {
		st.Publish([]*blendshape.Target{
			targetAt("a", V, float64(i)),
			targetAt("b", V, float64(i)),
			targetAt("c", V, float64(i)),
		}, V)
	}
	close(stop)
	wg.Wait()
}

func TestSnapshotImmutableAfterPublish(t *testing.T) {
	st := NewStore()
	src := targetAt("a", 2, 7)
	st.Publish([]*blendshape.Target{src}, 2)
	src.Deltas[0].X = 999 // mutate caller's slice
	tg, _ := st.Current().Get("a")
	if tg.Deltas[0].X != 7 {
		t.Fatal("published target must be immune to caller mutation")
	}
}
