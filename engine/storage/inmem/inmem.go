// Package inmem implements an engine storage backend using the a map-based key-value store.
package inmem

import (
	"github.com/micromdm/nanocmd/engine/storage/kv"
	"github.com/micromdm/nanocmd/utils/kv/kvmap"
	"github.com/micromdm/nanocmd/utils/uuid"
)

// InMem is an in-memory engine storage backend.
type InMem struct {
	*kv.KV
}

func New() *InMem {
	return &InMem{KV: kv.New(
		kvmap.NewBucket(),
		kvmap.NewBucket(),
		kvmap.NewBucket(),
		uuid.NewUUID(),
	)}
}
