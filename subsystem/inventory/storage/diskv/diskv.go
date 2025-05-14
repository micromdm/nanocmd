// Package diskv implements an inventory subsystem backend using diskv.
package diskv

import (
	"path/filepath"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvdiskv"
	"github.com/micromdm/nanolib/storage/kv/kvtxn"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is an inventory subsystem backend which uses diskv as the key-value store.
type Diskv struct {
	*kv.KV
}

// New creates a new profile store at on disk at path.
func New(path string) *Diskv {
	return &Diskv{
		KV: kv.New(kvtxn.New(kvdiskv.New(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "inventory"),
			Transform:    kvdiskv.FlatTransform,
			CacheSizeMax: 1024 * 1024,
		})))),
	}
}
