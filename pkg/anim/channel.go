package anim

// Channel is a playing instance of a clip with its own start time and gain.
// Multiple channels may drive the same targets concurrently; the engine merges
// their weights deterministically.
type Channel struct {
	Clip      *Clip
	Weight    float64
	StartTime float64
	Options   SampleOptions
}

// NewChannel creates a channel. NaN weight defaults to 1; negative weights
// become 0 and weights above 1 clamp to 1 (channels only add expression).
func NewChannel(c *Clip, startTime, weight float64, opts SampleOptions) *Channel {
	if weight != weight { // NaN
		weight = 1
	}
	if weight < 0 {
		weight = 0
	}
	if weight > 1 {
		weight = 1
	}
	return &Channel{Clip: c, Weight: weight, StartTime: startTime, Options: opts}
}

// WeightsAt returns this channel's target weights at absolute engine time t.
// Local clip time is t-StartTime sampled directly, so pausing/resuming and
// very long sessions never accumulate error.
func (ch *Channel) WeightsAt(t float64) map[string]float64 {
	if ch == nil || ch.Clip == nil || ch.Weight == 0 {
		return map[string]float64{}
	}
	local := t - ch.StartTime
	raw := SampleClip(ch.Clip, local, ch.Options)
	if ch.Weight == 1 {
		return raw
	}
	out := make(map[string]float64, len(raw))
	for name, v := range raw {
		out[name] = v * ch.Weight
	}
	return out
}

// ActiveAt reports whether the channel contributes at engine time t under
// LoopClamp (used to decide whether to keep a channel around).
func (ch *Channel) ActiveAt(t float64) bool {
	if ch == nil || ch.Clip == nil || ch.Weight == 0 {
		return false
	}
	if ch.Options.Loop != LoopClamp {
		return true
	}
	local := t - ch.StartTime
	return local >= 0 && local <= ch.Clip.Duration
}
