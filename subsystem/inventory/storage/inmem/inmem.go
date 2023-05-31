// Package inmem implements an in-memory inventory subsystem storage backend.
package inmem

import (
	"context"
	"sync"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
)

// InMem represents the in-memory enrollment inventory data store.
type InMem struct {
	mu  sync.RWMutex
	inv map[string]storage.Values
}

// New creates a new initialized inventory data store.
func New() *InMem {
	return &InMem{inv: make(map[string]storage.Values)}
}

// RetrieveInventory retrieves the inventory data for enrollment IDs.
func (s *InMem) RetrieveInventory(ctx context.Context, opt *storage.SearchOptions) (map[string]storage.Values, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if opt == nil || len(opt.IDs) <= 0 {
		return nil, nil
	}
	ret := make(map[string]storage.Values)
	for _, id := range opt.IDs {
		if vals, ok := s.inv[id]; ok {
			ret[id] = make(storage.Values)
			for k, v := range vals {
				ret[id][k] = v
			}
		}
	}
	return ret, nil
}

// StoreInventoryValues stores inventory data about the specified ID.
func (s *InMem) StoreInventoryValues(ctx context.Context, id string, values storage.Values) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inv[id] == nil {
		s.inv[id] = values
	} else {
		for k, v := range values {
			s.inv[id][k] = v
		}
	}
	return nil
}

// DeleteInventory deletes all inventory data for an enrollment ID.
func (s *InMem) DeleteInventory(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inv, id)
	return nil
}
