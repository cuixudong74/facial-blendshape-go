package engine

import (
	"sync"

	"github.com/cuixudong74/facial-blendshape-go/pkg/anim"
	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
	"github.com/cuixudong74/facial-blendshape-go/pkg/store"
)

// Engine drives layered evaluation from animation channels over a hot store.
//
// Concurrency: EvaluateAt holds no mutable per-frame state and is safe to call
// from any number of goroutines. The neutral mesh is owned and never mutated;
// the active targets are an immutable published snapshot; the only cached
// derived data (a sorted target slice keyed by snapshot version) is replaced
// atomically together with the snapshot pointer. Therefore every goroutine
// sees a fully consistent (snapshot, targets) pair and identical inputs yield
// bit-identical output regardless of concurrency.
type Engine struct {
	neutral []blendshape.Vec3
	policy  blendshape.WeightPolicy
	store   *store.Store
	mixer   *blendshape.LayerMixer

	targetsMu sync.RWMutex
	ver       uint64
	targets   []*blendshape.Target
}

// NewEngine constructs an engine. The neutral mesh is copied defensively.
func NewEngine(neutral []blendshape.Vec3, policy blendshape.WeightPolicy, st *store.Store) *Engine {
	n := make([]blendshape.Vec3, len(neutral))
	copy(n, neutral)
	e := &Engine{
		neutral: n,
		policy:  policy,
		store:   st,
	}
	e.mixer = blendshape.NewLayerMixerShared(n)
	return e
}

// VertexCount returns the rig vertex count.
func (e *Engine) VertexCount() int { return len(e.neutral) }

// Frame bundles one evaluation result.
type Frame struct {
	Vertices []blendshape.Vec3
	Version  uint64
	Issues   []blendshape.WeightIssue
	Errors   blendshape.ErrorList
}

// targetsFor returns the sorted target slice for a snapshot version. The
// sorted slice is allocated only when the published version changes, not per
// frame.
func (e *Engine) targetsFor(snap *store.Snapshot) []*blendshape.Target {
	e.targetsMu.RLock()
	if e.ver == snap.Version() && e.targets != nil {
		t := e.targets
		e.targetsMu.RUnlock()
		return t
	}
	e.targetsMu.RUnlock()

	e.targetsMu.Lock()
	defer e.targetsMu.Unlock()
	if e.ver != snap.Version() || e.targets == nil {
		e.targets = snap.All()
		e.ver = snap.Version()
	}
	return e.targets
}

// MergeChannelWeights combines channels into one raw weight map. Multiple
// channels on the same target add their contributions. Channel slice order
// fixes addition order (the caller expresses channel priority via ordering),
// and each channel's tracks iterate in sorted target order, so the sum is
// reproducible.
func MergeChannelWeights(channels []*anim.Channel, t float64) map[string]float64 {
	merged := make(map[string]float64)
	for _, ch := range channels {
		for name, v := range ch.WeightsAt(t) {
			merged[name] += v
		}
	}
	return merged
}

// EvaluateAt computes a frame at absolute time t. The snapshot is acquired
// exactly once, so a hot replacement landing mid-frame cannot tear the frame:
// it is wholly one published version.
func (e *Engine) EvaluateAt(channels []*anim.Channel, t float64, out []blendshape.Vec3) Frame {
	snap := e.store.Current()
	targets := e.targetsFor(snap)
	merged := MergeChannelWeights(channels, t)

	verts, issues, errs := e.mixer.Evaluate(targets, merged, e.policy, out)
	return Frame{Vertices: verts, Version: snap.Version(), Issues: issues, Errors: errs}
}

// EvaluateWeights evaluates explicit raw weights (no animation channels).
func (e *Engine) EvaluateWeights(weights map[string]float64, out []blendshape.Vec3) Frame {
	snap := e.store.Current()
	targets := e.targetsFor(snap)
	verts, issues, errs := e.mixer.Evaluate(targets, weights, e.policy, out)
	return Frame{Vertices: verts, Version: snap.Version(), Issues: issues, Errors: errs}
}

// ActiveChannels filters to channels still contributing at time t.
func ActiveChannels(channels []*anim.Channel, t float64) []*anim.Channel {
	out := make([]*anim.Channel, 0, len(channels))
	for _, ch := range channels {
		if ch.ActiveAt(t) {
			out = append(out, ch)
		}
	}
	return out
}
