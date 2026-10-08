package blendshape

import (
	"math"
	"sort"
)

// WeightPolicy defines how out-of-range or illegal weights are handled.
type WeightPolicy struct {
	Min         float64
	Max         float64
	Clamp       bool
	Normalize   bool
	SkipInvalid bool
}

// DefaultWeightPolicy clamps to [0,1] and renormalizes to sum 1.
func DefaultWeightPolicy() WeightPolicy {
	return WeightPolicy{
		Min:         0,
		Max:         1,
		Clamp:       true,
		Normalize:   true,
		SkipInvalid: true,
	}
}

// WeightIssue records a rejected or repaired weight, keyed by target name.
type WeightIssue struct {
	Target string
	Value  float64
	Reason string
}

func isFinite(v float64) bool {
	// math.IsNaN/IsInf keep this in one place.
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// NormalizeWeights validates, clamps and renormalizes a weight set.
//
// Processing is deterministic: inputs are repaired in sorted target-name order
// and the sum used for normalization is computed in that same order, so equal
// inputs always produce bit-identical outputs. Policies:
//
//   - NaN/+Inf/-Inf values are illegal. SkipInvalid drops them with an issue;
//     otherwise they are treated as 0 with an issue (so a single bad value can
//     never poison the whole frame).
//   - Values outside [Min,Max] are clamped when Clamp is set, otherwise the
//     weight is rejected (dropped with an issue).
//   - After repair, if Normalize is set and the positive sum is > 0, all
//     weights are scaled so that they sum to 1. An all-zero/empty input maps
//     deterministically to the zero weights instead of dividing by zero.
func NormalizeWeights(weights map[string]float64, policy WeightPolicy) (map[string]float64, []WeightIssue) {
	names := make([]string, 0, len(weights))
	for name := range weights {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make(map[string]float64, len(names))
	var issues []WeightIssue

	for _, name := range names {
		v := weights[name]
		if !isFinite(v) {
			issues = append(issues, WeightIssue{Target: name, Value: v,
				Reason: "weight is NaN or Inf"})
			if policy.SkipInvalid {
				continue
			}
			v = 0
		}
		if v < policy.Min {
			if policy.Clamp {
				issues = append(issues, WeightIssue{Target: name, Value: v,
					Reason: "weight below min, clamped"})
				v = policy.Min
			} else {
				issues = append(issues, WeightIssue{Target: name, Value: v,
					Reason: "weight below min, dropped"})
				continue
			}
		} else if v > policy.Max {
			if policy.Clamp {
				issues = append(issues, WeightIssue{Target: name, Value: v,
					Reason: "weight above max, clamped"})
				v = policy.Max
			} else {
				issues = append(issues, WeightIssue{Target: name, Value: v,
					Reason: "weight above max, dropped"})
				continue
			}
		}
		out[name] = v
	}

	if policy.Normalize {
		var sum float64
		for _, name := range names { // sorted order => reproducible summation
			if v, ok := out[name]; ok {
				sum += v
			}
		}
		if sum > 0 {
			for name := range out {
				out[name] = out[name] / sum
			}
		}
	}
	return out, issues
}
