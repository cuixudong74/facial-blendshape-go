package store

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTargetFile(t *testing.T, dir, name, kind string, n int, x float64) {
	t.Helper()
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, fileJSON(name, kind, n, x), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDirectoryReloader_HotReplaceAndReject(t *testing.T) {
	dir := t.TempDir()
	st := NewStore()
	writeTargetFile(t, dir, "a", "base", 2, 1)
	writeTargetFile(t, dir, "b", "base", 2, 1)
	r := NewDirectoryReloader(st, dir, 2, 0)

	changed, snap, errs := r.Scan()
	if !changed || len(errs) != 0 || snap.Size() != 2 {
		t.Fatalf("initial scan: changed=%v size=%d errs=%v", changed, snap.Size(), errs)
	}
	v := snap.Version()

	// rescan without changes
	changed2, _, _ := r.Scan()
	if changed2 {
		t.Fatal("unchanged directory must not republish")
	}

	// modify a, add c
	writeTargetFile(t, dir, "a", "base", 2, 5)
	writeTargetFile(t, dir, "c", "corrective", 2, 2)
	changed3, snap3, errs := r.Scan()
	if !changed3 || len(errs) != 0 || snap3.Version() != v+1 || snap3.Size() != 3 {
		t.Fatalf("hot replace: changed=%v ver=%d size=%d errs=%v",
			changed3, snap3.Version(), snap3.Size(), errs)
	}
	tg, _ := snap3.Get("a")
	if tg.Deltas[0].X != 5 {
		t.Fatal("replaced target content not updated")
	}

	// drop a corrupt file => whole scan rejected, old snapshot stays
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed4, snap4, errs := r.Scan()
	if changed4 || len(errs) == 0 || snap4.Version() != v+1 {
		t.Fatalf("corrupt dir must keep old snapshot: changed=%v ver=%d errs=%d",
			changed4, snap4.Version(), len(errs))
	}
}
