package config

import "testing"

func TestLoadFile_Testdata(t *testing.T) {
	c, err := LoadFile("../../testdata/config.json")
	if err != nil {
		t.Fatalf("load testdata: %v", err)
	}
	if c.Targets[0].Name != "smile" {
		t.Fatalf("unexpected target %q", c.Targets[0].Name)
	}
}
