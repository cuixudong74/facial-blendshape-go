package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
)

// TargetFile is the on-disk representation of one target.
type TargetFile struct {
	Name        string       `json:"name"`
	Kind        string       `json:"kind"`
	Priority    int          `json:"priority"`
	VertexCount int          `json:"vertexCount"`
	Deltas      [][3]float64 `json:"deltas"`
	Checksum    string       `json:"checksum,omitempty"`
}

var kindFromFile = map[string]blendshape.TargetKind{
	"base":       blendshape.KindBase,
	"additive":   blendshape.KindAdditive,
	"corrective": blendshape.KindCorrective,
}

// ChecksumTarget returns the hex SHA-256 of a target's semantic payload
// (name, kind and every delta coordinate). It is content-based and version
// independent.
func ChecksumTarget(t *blendshape.Target) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%d|%d|", t.Name, t.Kind, t.VertexCount)
	for _, d := range t.Deltas {
		fmt.Fprintf(h, "%.17g,%.17g,%.17g;", d.X, d.Y, d.Z)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// DecodeTarget parses and validates one file payload. When Checksum is present
// it must match the payload, otherwise the target is rejected as corrupt.
func DecodeTarget(f *TargetFile, vertexCount int) (*blendshape.Target, error) {
	if f == nil {
		return nil, fmt.Errorf("target payload is nil")
	}
	kind, ok := kindFromFile[f.Kind]
	if !ok {
		return nil, fmt.Errorf("unknown kind %q (want base|additive|corrective)", f.Kind)
	}
	if f.VertexCount != vertexCount {
		return nil, fmt.Errorf("vertex count mismatch: file has %d but rig has %d",
			f.VertexCount, vertexCount)
	}
	if len(f.Deltas) != vertexCount {
		return nil, fmt.Errorf("delta count %d does not match vertex count %d",
			len(f.Deltas), vertexCount)
	}
	deltas := make([]blendshape.Vec3, vertexCount)
	for i, d := range f.Deltas {
		deltas[i] = blendshape.Vec3{X: d[0], Y: d[1], Z: d[2]}
	}
	t := &blendshape.Target{
		Name:        f.Name,
		Kind:        kind,
		Priority:    f.Priority,
		VertexCount: vertexCount,
		Deltas:      deltas,
	}
	sum := ChecksumTarget(t)
	if f.Checksum != "" && f.Checksum != sum {
		return nil, fmt.Errorf("checksum mismatch: payload %s vs declared %s", sum, f.Checksum)
	}
	t.Checksum = sum
	if err := t.Validate(vertexCount); err != nil {
		return nil, err
	}
	return t, nil
}

// DecodeTargetJSON decodes a raw JSON document into one target.
func DecodeTargetJSON(raw []byte, vertexCount int) (*blendshape.Target, error) {
	var f TargetFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return DecodeTarget(&f, vertexCount)
}

// LoadTargets decodes many file payloads; corrupt entries are reported with
// their target name (or an ordinal when the name is unreadable) but never
// abort the batch, so one bad file cannot take down a hot reload.
func LoadTargets(files []*TargetFile, vertexCount int) ([]*blendshape.Target, blendshape.ErrorList) {
	out := make([]*blendshape.Target, 0, len(files))
	var errs blendshape.ErrorList
	for i, f := range files {
		t, err := DecodeTarget(f, vertexCount)
		if err != nil {
			name := fmt.Sprintf("<file#%d>", i)
			if f != nil && f.Name != "" {
				name = f.Name
			}
			errs = append(errs, &blendshape.TargetError{Target: name, Op: "load", Reason: err})
			continue
		}
		out = append(out, t)
	}
	return out, errs
}

// fileFromTarget converts a validated target back to its file representation.
func fileFromTarget(t *blendshape.Target) *TargetFile {
	deltas := make([][3]float64, len(t.Deltas))
	for i, d := range t.Deltas {
		deltas[i] = [3]float64{d.X, d.Y, d.Z}
	}
	kind := ""
	for k, v := range kindFromFile {
		if v == t.Kind {
			kind = k
		}
	}
	return &TargetFile{
		Name:        t.Name,
		Kind:        kind,
		Priority:    t.Priority,
		VertexCount: t.VertexCount,
		Deltas:      deltas,
		Checksum:    t.Checksum,
	}
}
