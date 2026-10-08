package anim

import (
	"math"
	"sort"
)

// Interpolation selects between-key interpolation.
type Interpolation int

const (
	StepInterp Interpolation = iota
	LinearInterp
)

// LoopMode selects timeline behavior past the clip duration.
type LoopMode int

const (
	// LoopClamp freezes the first/last key outside the clip range.
	LoopClamp LoopMode = iota
	// LoopRepeat wraps t modulo Duration.
	LoopRepeat
	// LoopPingPong bounces, alternating forward/backward each period.
	LoopPingPong
)

// SampleOptions configures sampling.
type SampleOptions struct {
	Interpolation Interpolation
	Loop          LoopMode
}

// DefaultSampleOptions is linear interpolation with repeating loop.
func DefaultSampleOptions() SampleOptions {
	return SampleOptions{Interpolation: LinearInterp, Loop: LoopRepeat}
}

// mapTime maps an absolute time onto the clip timeline. It is a pure function
// of (t, duration, mode): callers always feed the absolute engine clock rather
// than advancing a cursor incrementally, so running for hours produces the
// same value as a fresh sample at that time — no phase drift, no accumulated
// error. The integer cycle count is bounded by int64 to stay well-defined.
func mapTime(t, duration float64, mode LoopMode) float64 {
	switch mode {
	case LoopClamp:
		if t < 0 {
			return 0
		}
		if t > duration {
			return duration
		}
		return t
	case LoopPingPong:
		phase := math.Mod(t, 2*duration)
		if phase < 0 {
			phase += 2 * duration
		}
		if phase > duration {
			return 2*duration - phase
		}
		return phase
	default: // LoopRepeat
		phase := math.Mod(t, duration)
		if phase < 0 {
			phase += duration
		}
		// math.Mod can return duration when t is an exact multiple suffering
		// float rounding; pin it to the closed interval.
		if phase >= duration {
			phase = math.Nextafter(duration, 0)
		}
		return phase
	}
}

// SampleTrack evaluates a track at absolute timeline time t (seconds).
// Outside the key range the nearest endpoint value is held (step/linear agree
// there). Keys must be sorted ascending; callers obtain tracks from validated
// clips.
func SampleTrack(tr *Track, t float64, opts SampleOptions) float64 {
	if tr == nil || len(tr.Keys) == 0 {
		return 0
	}
	keys := tr.Keys
	duration := keys[len(keys)-1].Time
	phase := mapTime(t, duration, opts.Loop)

	// Find i such that keys[i].Time <= phase <= keys[i+1].Time.
	i := sort.Search(len(keys)-1, func(j int) bool { return keys[j+1].Time >= phase })
	if i >= len(keys)-1 {
		return keys[len(keys)-1].Value
	}
	if phase <= keys[0].Time {
		return keys[0].Value
	}
	k0, k1 := keys[i], keys[i+1]
	if opts.Interpolation == StepInterp {
		return k0.Value
	}
	span := k1.Time - k0.Time
	if span == 0 {
		return k1.Value
	}
	u := (phase - k0.Time) / span
	return k0.Value + (k1.Value-k0.Value)*u
}

// SampleClip evaluates every track at time t. Zero-valued tracks are omitted,
// so the mixer only sees active targets.
func SampleClip(c *Clip, t float64, opts SampleOptions) map[string]float64 {
	out := make(map[string]float64, len(c.Tracks))
	for _, tr := range c.SortedTracks() { // sorted iteration => stable construction
		v := SampleTrack(tr, t, opts)
		if v != 0 {
			out[tr.Target] = v
		}
	}
	return out
}
