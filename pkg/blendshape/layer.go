package blendshape

import "sort"

// LayerMixer combines targets in three deterministic layers:
//
//  1. base: the dominant expression shapes. Their weights are clamped and
//     renormalized to sum 1 within the base layer, then blended once.
//  2. additive: secondary shapes layered on top of the base result. Each is
//     clamped but applied with its own (non-renormalized) weight.
//  3. corrective: fix-up shapes (e.g. lip-compression on a combined smile +
//     jaw-open). Applied last so they always see the fully posed face.
//
// Within a layer targets apply in ascending Priority then Name order, removing
// any order ambiguity: float summation order is fixed and the result is
// reproducible regardless of the input slice order.
//
// A LayerMixer is stateless across evaluations: the neutral mesh and targets
// are read-only inputs (targets come from an immutable published snapshot), so
// a single mixer may serve any number of goroutines. Per-frame scratch is
// local and the only vertex-sized allocation is the output buffer, which the
// caller is encouraged to reuse.
type LayerMixer struct {
	neutral []Vec3
	targets []*Target
}

// NewLayerMixer creates a layered mixer over a private copy of neutral.
func NewLayerMixer(neutral []Vec3) *LayerMixer {
	n := make([]Vec3, len(neutral))
	copy(n, neutral)
	return &LayerMixer{neutral: n}
}

// NewLayerMixerShared creates a mixer reading neutral without copying. The
// caller guarantees neutral is never mutated afterwards.
func NewLayerMixerShared(neutral []Vec3) *LayerMixer { return &LayerMixer{neutral: neutral} }

// SetTargets installs the default target set used by EvaluateInstalled. Only
// the immutable slice reference is retained.
func (m *LayerMixer) SetTargets(targets []*Target) { m.targets = targets }

// VertexCount returns the rig vertex count.
func (m *LayerMixer) VertexCount() int { return len(m.neutral) }

type layerGroup struct {
	targets []*Target
	weights map[string]float64
}

// Evaluate produces one frame from the explicit targets slice.
func (m *LayerMixer) Evaluate(targets []*Target, weights map[string]float64, policy WeightPolicy, out []Vec3) ([]Vec3, []WeightIssue, ErrorList) {
	return m.evaluate(targets, weights, policy, out)
}

// EvaluateInstalled evaluates against the targets supplied via SetTargets.
func (m *LayerMixer) EvaluateInstalled(weights map[string]float64, policy WeightPolicy, out []Vec3) ([]Vec3, []WeightIssue, ErrorList) {
	return m.evaluate(m.targets, weights, policy, out)
}

func (m *LayerMixer) evaluate(targets []*Target, weights map[string]float64, policy WeightPolicy, out []Vec3) ([]Vec3, []WeightIssue, ErrorList) {
	groups := [3]layerGroup{
		{weights: map[string]float64{}},
		{weights: map[string]float64{}},
		{weights: map[string]float64{}},
	}

	var errs ErrorList
	known := make(map[string]struct{})
	for _, t := range targets {
		if err := t.Validate(len(m.neutral)); err != nil {
			errs = append(errs, &TargetError{Target: safeName(t), Op: "layer-eval", Reason: err})
			continue
		}
		known[t.Name] = struct{}{}
		g := &groups[int(t.Kind)-1]
		g.targets = append(g.targets, t)
		if w, ok := weights[t.Name]; ok {
			g.weights[t.Name] = w
		}
	}

	var issues []WeightIssue
	for name, v := range weights {
		if _, ok := known[name]; !ok {
			issues = append(issues, WeightIssue{Target: name, Value: v,
				Reason: "unknown target, ignored"})
		}
	}

	buf := out
	if len(buf) != len(m.neutral) {
		buf = make([]Vec3, len(m.neutral))
	}
	copy(buf, m.neutral)

	for gi := range groups {
		// Only the base layer renormalizes across its members; additive and
		// corrective weights retain their independent magnitude.
		lp := policy
		lp.Normalize = policy.Normalize && gi == 0
		nw, li := NormalizeWeights(groups[gi].weights, lp)
		issues = append(issues, li...)

		sorted := groups[gi].targets
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].Priority != sorted[j].Priority {
				return sorted[i].Priority < sorted[j].Priority
			}
			return sorted[i].Name < sorted[j].Name
		})
		for _, t := range sorted {
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
	}
	return buf, issues, errs
}
