package store

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
)

// Snapshot is an immutable, fully-validated set of targets at one version.
// Targets inside are deep copies, so a snapshot is safe for unlimited
// concurrent readers and can never be mutated after publication.
type Snapshot struct {
	version uint64
	targets map[string]*blendshape.Target
}

// Version returns the monotonically increasing snapshot version.
func (s *Snapshot) Version() uint64 { return s.version }

// Get looks up a target.
func (s *Snapshot) Get(name string) (*blendshape.Target, bool) {
	t, ok := s.targets[name]
	return t, ok
}

// All returns targets sorted by name (a fresh slice; targets themselves are
// shared and must not be mutated).
func (s *Snapshot) All() []*blendshape.Target {
	out := make([]*blendshape.Target, 0, len(s.targets))
	for _, t := range s.targets {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns sorted target names.
func (s *Snapshot) Names() []string {
	out := make([]string, 0, len(s.targets))
	for name := range s.targets {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Size returns the number of targets in the snapshot.
func (s *Snapshot) Size() int { return len(s.targets) }

// Store publishes target snapshots atomically. Readers always observe either
// the old complete set or the new complete set, never a half-built one:
// a replacement is fully decoded, cloned and validated into a private map
// before a single atomic.Pointer.Store swaps it in.
type Store struct {
	current atomic.Pointer[Snapshot]
	buildMu sync.Mutex
}

// NewStore creates an empty store.
func NewStore() *Store {
	s := &Store{}
	s.current.Store(&Snapshot{version: 0, targets: map[string]*blendshape.Target{}})
	return s
}

// Current returns the active snapshot (never nil).
func (s *Store) Current() *Snapshot { return s.current.Load() }

// Publish validates and atomically installs a new complete snapshot built
// from targets. Any invalid target aborts publication and the previous
// snapshot stays live; the returned errors identify each offending target.
func (s *Store) Publish(targets []*blendshape.Target, vertexCount int) (*Snapshot, blendshape.ErrorList) {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()
	return s.commit(targets, vertexCount)
}

// Replace atomically upserts and removes named targets on top of the current
// snapshot. Either the whole change applies or none of it does.
func (s *Store) Replace(upserts []*blendshape.Target, removes []string, vertexCount int) (*Snapshot, blendshape.ErrorList) {
	s.buildMu.Lock()
	defer s.buildMu.Unlock()

	cur := s.current.Load()
	remove := make(map[string]struct{}, len(removes))
	for _, n := range removes {
		remove[n] = struct{}{}
	}
	merged := make(map[string]*blendshape.Target, cur.Size()+len(upserts))
	for _, t := range cur.All() {
		if _, drop := remove[t.Name]; !drop {
			merged[t.Name] = t
		}
	}
	for _, t := range upserts { // upsert: new payload wins
		merged[t.Name] = t
	}
	base := make([]*blendshape.Target, 0, len(merged))
	for _, t := range merged {
		base = append(base, t)
	}
	return s.commit(base, vertexCount)
}

// commit builds and validates a snapshot entirely before the atomic swap.
func (s *Store) commit(targets []*blendshape.Target, vertexCount int) (*Snapshot, blendshape.ErrorList) {
	next := make(map[string]*blendshape.Target, len(targets))
	var errs blendshape.ErrorList
	for _, t := range targets {
		if t == nil {
			errs = append(errs, &blendshape.TargetError{Target: "<nil>", Op: "publish",
				Reason: errNilTarget()})
			continue
		}
		if err := t.Validate(vertexCount); err != nil {
			errs = append(errs, &blendshape.TargetError{Target: t.Name, Op: "publish", Reason: err})
			continue
		}
		if _, dup := next[t.Name]; dup {
			errs = append(errs, &blendshape.TargetError{Target: t.Name, Op: "publish",
				Reason: errDuplicateTarget()})
			continue
		}
		// Defensive deep copy: publication hands ownership of the bytes to
		// the store; later caller mutation cannot affect a live frame.
		next[t.Name] = t.Clone()
	}
	if len(errs) > 0 {
		return s.current.Load(), errs
	}
	prev := s.current.Load().version
	snap := &Snapshot{version: prev + 1, targets: next}
	s.current.Store(snap) // publish: single atomic point
	return snap, nil
}
