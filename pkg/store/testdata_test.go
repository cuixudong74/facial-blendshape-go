package store

import (
	"os"
	"testing"
)

func TestDecodeTestdataSmile(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/smile.json")
	if err != nil {
		t.Fatal(err)
	}
	tg, derr := DecodeTargetJSON(raw, 3)
	if derr != nil {
		t.Fatalf("decode: %v", derr)
	}
	if tg.Name != "smile" || tg.Checksum == "" || tg.Deltas[1].Y != 1 {
		t.Fatalf("decoded wrong: %+v", tg)
	}
}
