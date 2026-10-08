package config

import "testing"

const validCfg = `{
  "vertexCount": 2,
  "weights": {"min": 0, "max": 1, "clamp": true, "normalize": true, "skipInvalid": true},
  "targets": [
    {"name": "smile", "file": "smile.json", "kind": "base", "priority": 0}
  ],
  "clips": [
    {"name": "talk", "duration": 1, "tracks": [
      {"target": "smile", "keys": [{"time": 0, "value": 0}, {"time": 1, "value": 1}]}
    ]}
  ]
}`

func TestLoad_OK(t *testing.T) {
	c, err := Load([]byte(validCfg))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.VertexCount != 2 || len(c.Targets) != 1 || len(c.Clips) != 1 {
		t.Fatalf("parsed wrong: %+v", c)
	}
	if !c.Weights.Policy().Clamp {
		t.Fatal("policy conversion wrong")
	}
	clips, err := c.BuildClips()
	if err != nil || len(clips) != 1 || clips[0].Duration != 1 {
		t.Fatalf("build clips: %v %v", clips, err)
	}
}

func TestLoad_BadJSON(t *testing.T) {
	if _, err := Load([]byte("{oops")); err == nil {
		t.Fatal("bad JSON must error")
	}
}

func TestValidate_Failures(t *testing.T) {
	cases := map[string]string{
		"bad vertex count": `{"vertexCount": 0}`,
		"min>max":          `{"vertexCount": 2, "weights": {"min": 1, "max": 0}}`,
		"bad kind":         `{"vertexCount": 2, "targets": [{"name":"x","kind":"nope"}]}`,
		"dup target":       `{"vertexCount": 2, "targets": [{"name":"x","kind":"base"},{"name":"x","kind":"base"}]}`,
		"bad clip":         `{"vertexCount": 2, "clips": [{"name":"c","duration":1,"tracks":[{"target":"t","keys":[{"time":2},{"time":1}]}]}]}`,
	}
	for name, body := range cases {
		if _, err := Load([]byte(body)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
