// Package inmem implements a command plan storage backend backed by an in-memory key-valye store.
package inmem

import (
	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
)

// InMem is a command plan storage backend backed by an in-memory key-valye store.
type InMem struct {
	*kv.KV
}

func New() *InMem {
	return &InMem{KV: kv.New(kvmap.New())}
}
