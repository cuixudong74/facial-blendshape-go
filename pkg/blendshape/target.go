package blendshape

import (
	"errors"
	"strconv"
)

// TargetKind identifies the layer a target belongs to.
type TargetKind int

const (
	KindUnknown TargetKind = iota
	KindBase
	KindAdditive
	KindCorrective
)

// Target is a single morph target: per-vertex deltas relative to the neutral
// mesh. A Target is immutable once published by the store.
type Target struct {
	Name        string
	Kind        TargetKind
	Priority    int
	VertexCount int
	Deltas      []Vec3
	Checksum    string
}

// Validate checks structural integrity against the rig vertex count.
func (t *Target) Validate(vertexCount int) error {
	if t == nil {
		return errors.New("target is nil")
	}
	if t.Name == "" {
		return errors.New("target name is empty")
	}
	switch t.Kind {
	case KindBase, KindAdditive, KindCorrective:
	default:
		return errors.New("unknown target kind")
	}
	if t.VertexCount != vertexCount {
		return errors.New("vertex count mismatch: target has " +
			strconv.Itoa(t.VertexCount) + " but rig has " + strconv.Itoa(vertexCount))
	}
	if len(t.Deltas) != vertexCount {
		return errors.New("delta slice length does not match vertex count")
	}
	for i, d := range t.Deltas {
		if !isFinite(d.X) || !isFinite(d.Y) || !isFinite(d.Z) {
			return errors.New("delta at vertex " + strconv.Itoa(i) + " is NaN/Inf")
		}
	}
	return nil
}

// Clone returns a deep copy so callers cannot mutate published targets.
func (t *Target) Clone() *Target {
	if t == nil {
		return nil
	}
	deltas := make([]Vec3, len(t.Deltas))
	copy(deltas, t.Deltas)
	return &Target{
		Name:        t.Name,
		Kind:        t.Kind,
		Priority:    t.Priority,
		VertexCount: t.VertexCount,
		Deltas:      deltas,
		Checksum:    t.Checksum,
	}
}
