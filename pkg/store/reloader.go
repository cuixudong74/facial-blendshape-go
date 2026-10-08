package store

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/cuixudong74/facial-blendshape-go/pkg/blendshape"
)

// DirectoryReloader polls a directory of *.json target files and republishes
// the whole directory atomically whenever it changes. A failed parse/validate
// cycle keeps the previous snapshot live, so driving animations never observe
// a half-written directory.
type DirectoryReloader struct {
	store       *Store
	dir         string
	vertexCount int
	interval    time.Duration
	signature   map[string]string // target name -> last published checksum
}

// NewDirectoryReloader creates a reloader polling dir every interval.
func NewDirectoryReloader(st *Store, dir string, vertexCount int, interval time.Duration) *DirectoryReloader {
	if interval <= 0 {
		interval = time.Second
	}
	return &DirectoryReloader{
		store:       st,
		dir:         dir,
		vertexCount: vertexCount,
		interval:    interval,
		signature:   map[string]string{},
	}
}

// Scan reads the directory once and republishes if content changed. It returns
// whether a new snapshot was published. On any corrupt/illegal file the scan is
// rejected wholesale and the previous snapshot stays live.
func (r *DirectoryReloader) Scan() (bool, *Snapshot, blendshape.ErrorList) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return false, r.store.Current(), blendshape.ErrorList{
			&blendshape.TargetError{Target: "directory", Op: "scan", Reason: err},
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var files []*TargetFile
	sigs := map[string]string{}
	var errs blendshape.ErrorList
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(r.dir, e.Name()))
		if err != nil {
			errs = append(errs, &blendshape.TargetError{Target: e.Name(), Op: "read", Reason: err})
			continue
		}
		t, err := DecodeTargetJSON(raw, r.vertexCount)
		if err != nil {
			errs = append(errs, &blendshape.TargetError{Target: e.Name(), Op: "decode", Reason: err})
			continue
		}
		files = append(files, fileFromTarget(t))
		sigs[t.Name] = t.Checksum
	}
	if len(errs) > 0 {
		return false, r.store.Current(), errs
	}
	if sameSignatures(r.signature, sigs) && r.store.Current().Size() == len(sigs) {
		return false, r.store.Current(), nil
	}
	targets, loadErrs := LoadTargets(files, r.vertexCount)
	if len(loadErrs) > 0 {
		return false, r.store.Current(), loadErrs
	}
	snap, pubErrs := r.store.Publish(targets, r.vertexCount)
	if len(pubErrs) > 0 {
		return false, r.store.Current(), pubErrs
	}
	r.signature = sigs
	return true, snap, nil
}

// Run polls until the context is cancelled.
func (r *DirectoryReloader) Run(ctx context.Context, onReload func(changed bool, snap *Snapshot, errs error)) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, snap, errs := r.Scan()
			if onReload != nil {
				onReload(changed, snap, errs.ErrOrNil())
			}
		}
	}
}

func sameSignatures(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
