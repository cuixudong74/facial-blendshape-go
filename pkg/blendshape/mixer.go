package blendshape

import "sort"

// Mixer performs weighted mixing of morph targets against a fixed neutral mesh.
// It is stateless across calls and safe for concurrent use; the neutral mesh
// and targets are read-only and the only vertex-sized allocation is the output
// buffer, which callers reuse frame to frame.
type Mixer struct {
	neutral []Vec3
}

// NewMixer creates a mixer for the given neutral mesh (copied defensively).
func NewMixer(neutral []Vec3) *Mixer {
	n := make([]Vec3, len(neutral))
	copy(n, neutral)
	return &Mixer{neutral: n}
}

// NewMixerShared creates a mixer reading neutral without copying; the caller
// guarantees neutral is never mutated afterwards.
func NewMixerShared(neutral []Vec3) *Mixer { return &Mixer{neutral: neutral} }

// VertexCount returns the rig vertex count.
func (m *Mixer) VertexCount() int { return len(m.neutral) }

// Mix blends targets with the given weights. Invalid targets are reported and
// skipped; the result is written into out and returned. If out is nil or has
// the wrong length a single new output buffer is allocated.
//
// Cost is O(V*T) (V vertices, T active targets) with exactly one output-sized
// write; targets are visited in sorted name order so summation order, and
// therefore floating-point results, are reproducible.
func (m *Mixer) Mix(targets []*Target, weights map[string]float64, policy WeightPolicy, out []Vec3) ([]Vec3, []WeightIssue, ErrorList) {
	valid := make([]*Target, 0, len(targets))
	var errs ErrorList
	known := make(map[string]float64, len(weights))

	for _, t := range targets {
		if err := t.Validate(len(m.neutral)); err != nil {
			errs = append(errs, &TargetError{Target: safeName(t), Op: "mix", Reason: err})
			continue
		}
		if w, ok := weights[t.Name]; ok {
			known[t.Name] = w
		}
		valid = append(valid, t)
	}

	var issues []WeightIssue
	for name := range weights {
		if _, ok := known[name]; !ok {
			issues = append(issues, WeightIssue{Target: name,
				Value: weights[name], Reason: "unknown target, ignored"})
		}
	}

	nw, ni := NormalizeWeights(known, policy)
	issues = append(issues, ni...)

	sort.SliceStable(valid, func(i, j int) bool { return valid[i].Name < valid[j].Name })

	buf := out
	if len(buf) != len(m.neutral) {
		buf = make([]Vec3, len(m.neutral))
	}
	copy(buf, m.neutral)
	for _, t := range valid {
		w := nw[t.Name]
		if w == 0 {
			continue
		}
		d := t.Deltas
		for i := range buf {
			buf[i].X += w * d[i].X
			buf[i].Y += w * d[i].Y
			buf[i].Z += w * d[i].Z
		}
	}
	return buf, issues, errs
}

func safeName(t *Target) string {
	if t == nil || t.Name == "" {
		return "<unnamed>"
	}
	return t.Name
}
