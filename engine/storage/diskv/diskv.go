// Package diskv implements an engine storage backend using the diskv key-value store.
package diskv

import (
	"path/filepath"

	"github.com/micromdm/nanocmd/engine/storage/kv"
	"github.com/micromdm/nanocmd/utils/uuid"

	"github.com/micromdm/nanolib/storage/kv/kvdiskv"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is a a diskv-backed engine storage backend.
type Diskv struct {
	*kv.KV
}

func New(path string) *Diskv {
	flatTransform := func(s string) []string { return []string{} }
	return &Diskv{KV: kv.New(
		kvdiskv.New(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "engine", "step"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		})),
		kvdiskv.New(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "engine", "idcmd"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		})),
		kvdiskv.New(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "engine", "eventsubs"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		})),
		uuid.NewUUID(),
		kvdiskv.New(diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "engine", "wfstatus"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		})),
	)}
}
