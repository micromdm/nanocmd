// Package diskv implements a diskv-backed FileVault storage backend.
package diskv

import (
	"context"
	"path/filepath"

	"github.com/micromdm/nanocmd/subsystem/filevault/storage"
	"github.com/micromdm/nanocmd/subsystem/filevault/storage/kv"
	"github.com/micromdm/nanocmd/utils/kv/kvdiskv"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv implements a diskv-backed FileVault storage backend.
type Diskv struct {
	*kv.KV
}

func New(path string, p storage.PRKStorage) (*Diskv, error) {
	flatTransform := func(s string) []string { return []string{} }
	kvStore, err := kv.New(
		context.Background(),
		kvdiskv.NewBucket(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "fvkey"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		})),
		p,
	)
	return &Diskv{KV: kvStore}, err
}
