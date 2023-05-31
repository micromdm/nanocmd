// Package inmem implements a command plan storage backend backed by an on-disk key-valye store.
package diskv

import (
	"path/filepath"

	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage/kv"
	"github.com/micromdm/nanocmd/utils/kv/kvdiskv"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is a command plan storage backend backed by an on-disk key-valye store.
type Diskv struct {
	*kv.KV
}

// New creates a new initialized CMDPlan data store.
func New(path string) *Diskv {
	flatTransform := func(s string) []string { return []string{} }
	return &Diskv{
		KV: kv.New(kvdiskv.NewBucket(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "cmdplan"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		}))),
	}
}
