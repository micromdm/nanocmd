// Package inmem implements an in-memory inventory subsystem storage backend.
package inmem

import (
	"github.com/micromdm/nanocmd/subsystem/inventory/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
)

// InMem is an in-memory inventory subsystem storage system backend.
type InMem struct {
	*kv.KV
}

// New creates a new inventory subsystem storage system backend.
func New() *InMem {
	return &InMem{KV: kv.New(kvmap.New())}
}
