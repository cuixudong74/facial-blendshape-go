package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cuixudong74/facial-blendshape-go/pkg/anim"
	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
)

// Config is the root runtime configuration.
type Config struct {
	VertexCount int              `json:"vertexCount"`
	Weights     WeightConfig     `json:"weights"`
	Targets     []TargetSource   `json:"targets"`
	Clips       []ClipDefinition `json:"clips"`
}

// WeightConfig maps to blendshape.WeightPolicy.
type WeightConfig struct {
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Clamp       bool    `json:"clamp"`
	Normalize   bool    `json:"normalize"`
	SkipInvalid bool    `json:"skipInvalid"`
}

// Policy converts the config into a runtime WeightPolicy.
func (w WeightConfig) Policy() blendshape.WeightPolicy {
	return blendshape.WeightPolicy{
		Min:         w.Min,
		Max:         w.Max,
		Clamp:       w.Clamp,
		Normalize:   w.Normalize,
		SkipInvalid: w.SkipInvalid,
	}
}

// TargetSource points at one deformable target payload.
type TargetSource struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	Kind     string `json:"kind"`
	Priority int    `json:"priority"`
}

// ClipDefinition is an animation clip definition.
type ClipDefinition struct {
	Name     string            `json:"name"`
	Duration float64           `json:"duration"`
	Tracks   []TrackDefinition `json:"tracks"`
}

// TrackDefinition is one animated target.
type TrackDefinition struct {
	Target string          `json:"target"`
	Keys   []anim.Keyframe `json:"keys"`
}

// BuildClip validates and converts a clip definition.
func (d ClipDefinition) BuildClip() (*anim.Clip, error) {
	c := &anim.Clip{
		Name:     d.Name,
		Duration: d.Duration,
		Tracks:   make([]*anim.Track, 0, len(d.Tracks)),
	}
	for _, td := range d.Tracks {
		c.Tracks = append(c.Tracks, &anim.Track{Target: td.Target, Keys: td.Keys})
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("clip %q: %w", d.Name, err)
	}
	return c, nil
}

// BuildClips converts all clip definitions, failing on the first invalid one.
func (c *Config) BuildClips() ([]*anim.Clip, error) {
	out := make([]*anim.Clip, 0, len(c.Clips))
	seen := map[string]struct{}{}
	for _, d := range c.Clips {
		if _, dup := seen[d.Name]; dup {
			return nil, fmt.Errorf("duplicate clip name %q", d.Name)
		}
		seen[d.Name] = struct{}{}
		clip, err := d.BuildClip()
		if err != nil {
			return nil, err
		}
		out = append(out, clip)
	}
	return out, nil
}

// Validate checks cross-field consistency.
func (c *Config) Validate() error {
	if c.VertexCount <= 0 {
		return fmt.Errorf("vertexCount must be positive, got %d", c.VertexCount)
	}
	if c.Weights.Min > c.Weights.Max {
		return fmt.Errorf("weight min %v exceeds max %v", c.Weights.Min, c.Weights.Max)
	}
	names := map[string]struct{}{}
	for _, ts := range c.Targets {
		if ts.Name == "" {
			return fmt.Errorf("target source with empty name")
		}
		if _, dup := names[ts.Name]; dup {
			return fmt.Errorf("duplicate target source %q", ts.Name)
		}
		names[ts.Name] = struct{}{}
		switch ts.Kind {
		case "base", "additive", "corrective":
		default:
			return fmt.Errorf("target %q has unknown kind %q", ts.Name, ts.Kind)
		}
	}
	if _, err := c.BuildClips(); err != nil {
		return err
	}
	return nil
}

// Load reads and validates a JSON config.
func Load(data []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid config JSON: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// LoadFile reads and validates a JSON config file.
func LoadFile(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(raw)
}
