// Package inmem implements an engine storage backend using the a map-based key-value store.
package inmem

import (
	"github.com/micromdm/nanocmd/engine/storage/kv"
	"github.com/micromdm/nanocmd/utils/uuid"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
)

// InMem is an in-memory engine storage backend.
type InMem struct {
	*kv.KV
}

func New() *InMem {
	return &InMem{KV: kv.New(
		kvmap.New(),
		kvmap.New(),
		kvmap.New(),
		uuid.NewUUID(),
		kvmap.New(),
	)}
}
