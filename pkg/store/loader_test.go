package store

import (
	"encoding/json"
	"testing"

	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
)

func fileJSON(name, kind string, n int, x float64) []byte {
	deltas := make([][3]float64, n)
	for i := range deltas {
		deltas[i] = [3]float64{x, 0, 0}
	}
	f := TargetFile{Name: name, Kind: kind, VertexCount: n, Deltas: deltas}
	raw, _ := json.Marshal(f)
	return raw
}

func TestDecodeTargetJSON_OK(t *testing.T) {
	tg, err := DecodeTargetJSON(fileJSON("smile", "base", 3, 1), 3)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if tg.Kind != blendshape.KindBase || tg.Checksum == "" {
		t.Fatalf("decoded target wrong: %+v", tg)
	}
}

func TestDecodeTargetJSON_BadJSON(t *testing.T) {
	if _, err := DecodeTargetJSON([]byte("{nope"), 3); err == nil {
		t.Fatal("corrupt JSON must error")
	}
}

func TestDecodeTargetJSON_DimensionMismatch(t *testing.T) {
	if _, err := DecodeTargetJSON(fileJSON("smile", "base", 3, 1), 4); err == nil {
		t.Fatal("dimension mismatch must error")
	}
	f := TargetFile{Name: "x", Kind: "base", VertexCount: 3, Deltas: [][3]float64{{1}}}
	raw, _ := json.Marshal(f)
	if _, err := DecodeTargetJSON(raw, 3); err == nil {
		t.Fatal("delta count mismatch must error")
	}
}

func TestDecodeTargetJSON_ChecksumTamper(t *testing.T) {
	good, err := DecodeTargetJSON(fileJSON("smile", "base", 2, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	// tamper: declare the good checksum but change a delta.
	deltas := make([][3]float64, 2)
	deltas[0] = [3]float64{9, 9, 9}
	bad := TargetFile{Name: "smile", Kind: "base", VertexCount: 2,
		Deltas: deltas, Checksum: good.Checksum}
	raw, _ := json.Marshal(bad)
	if _, err := DecodeTargetJSON(raw, 2); err == nil {
		t.Fatal("checksum tampering must be rejected")
	}
}

func TestLoadTargets_PartialCorruptionIsolates(t *testing.T) {
	f1 := &TargetFile{}
	_ = json.Unmarshal(fileJSON("good", "additive", 2, 1), f1)
	f2 := &TargetFile{}
	_ = json.Unmarshal(fileJSON("bad", "weird-kind", 2, 1), f2)

	targets, errs := LoadTargets([]*TargetFile{f1, f2}, 2)
	if len(targets) != 1 || targets[0].Name != "good" {
		t.Fatalf("good target must survive, got %v", targets)
	}
	if len(errs) != 1 {
		t.Fatalf("want 1 error got %d", len(errs))
	}
	if te := errs[0].(*blendshape.TargetError); te.Target != "bad" {
		t.Fatalf("error must identify bad: %v", te)
	}
}
