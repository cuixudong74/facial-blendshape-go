package anim

import (
	"errors"
	"math"
	"sort"
	"strconv"
)

// Keyframe is one (time, value) pair on a track, in seconds.
type Keyframe struct {
	Time  float64
	Value float64
}

// Track animates a single blendshape target.
type Track struct {
	Target string
	Keys   []Keyframe
}

// Clip is a collection of tracks sharing one timeline.
// Duration is a property of the clip; sampling is continuous in time and is
// independent of any fixed sample rate.
type Clip struct {
	Name     string
	Duration float64
	Tracks   []*Track
}

// Validate checks timeline consistency: strictly increasing finite key times
// inside [0, Duration], finite values, unique target names.
func (c *Clip) Validate() error {
	if c == nil {
		return errors.New("clip is nil")
	}
	if !(c.Duration > 0) || math.IsNaN(c.Duration) || math.IsInf(c.Duration, 0) {
		return errors.New("clip duration must be positive and finite")
	}
	seen := make(map[string]struct{})
	for _, tr := range c.Tracks {
		if tr == nil {
			return errors.New("nil track")
		}
		if tr.Target == "" {
			return errors.New("track has empty target")
		}
		if _, dup := seen[tr.Target]; dup {
			return errors.New("duplicate track for target " + tr.Target)
		}
		seen[tr.Target] = struct{}{}
		if len(tr.Keys) == 0 {
			return errors.New("track " + tr.Target + " has no keyframes")
		}
		prev := math.Inf(-1)
		for i, k := range tr.Keys {
			if math.IsNaN(k.Time) || math.IsInf(k.Time, 0) {
				return errors.New("track " + tr.Target + ": non-finite key time")
			}
			if math.IsNaN(k.Value) || math.IsInf(k.Value, 0) {
				return errors.New("track " + tr.Target + ": non-finite key value")
			}
			if k.Time < prev {
				return errors.New("track " + tr.Target + ": keyframes not sorted at index " +
					strconv.Itoa(i))
			}
			if k.Time == prev {
				return errors.New("track " + tr.Target + ": duplicate key time at index " +
					strconv.Itoa(i))
			}
			prev = k.Time
		}
	}
	return nil
}

// SortedTracks returns tracks ordered by target name for deterministic output.
func (c *Clip) SortedTracks() []*Track {
	out := make([]*Track, len(c.Tracks))
	copy(out, c.Tracks)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Target < out[j].Target })
	return out
}

// Clone deep-copies the clip.
func (c *Clip) Clone() *Clip {
	if c == nil {
		return nil
	}
	tracks := make([]*Track, len(c.Tracks))
	for i, tr := range c.Tracks {
		keys := make([]Keyframe, len(tr.Keys))
		copy(keys, tr.Keys)
		tracks[i] = &Track{Target: tr.Target, Keys: keys}
	}
	return &Clip{Name: c.Name, Duration: c.Duration, Tracks: tracks}
}
