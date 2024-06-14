// Package diskv implements a storage backend for the Profile subsystem backed by diskv.
package diskv

import (
	"path/filepath"

	"github.com/micromdm/nanocmd/subsystem/profile/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvdiskv"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is a profile storage backend that uses an on-disk key-valye store.
type Diskv struct {
	*kv.KV
}

// New creates a new profile store at on disk at path.
func New(path string) *Diskv {
	return &Diskv{
		KV: kv.New(kvdiskv.New(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "profile"),
			Transform:    kvdiskv.FlatTransform,
			CacheSizeMax: 1024 * 1024,
		}))),
	}
}
