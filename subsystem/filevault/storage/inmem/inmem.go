// Package diskv implements an in-memory FileVault storage backend.
package inmem

import (
	"context"

	"github.com/micromdm/nanocmd/subsystem/filevault/storage"
	"github.com/micromdm/nanocmd/subsystem/filevault/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
)

// InMem implements an in-memory FileVault storage backend.
type InMem struct {
	*kv.KV
}

func New(p storage.PRKStorage) (*InMem, error) {
	kvStore, err := kv.New(context.Background(), kvmap.New(), p)
	return &InMem{KV: kvStore}, err
}
