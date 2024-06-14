// Package inmem implements an in-memory storage backend for the Profile subsystem.
package inmem

import (
	"github.com/micromdm/nanocmd/subsystem/profile/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
)

// InMem is a profile storage backend using an in-memory key-valye store.
type InMem struct {
	*kv.KV
}

func New() *InMem {
	return &InMem{KV: kv.New(kvmap.New())}
}
