// Package kv implements an inventory subsystem storage backend using a key-value store.
package kv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

// KV is an inventory subsystem storage backend using a key-value store.
type KV struct {
	b kv.KeysPrefixTraversingBucket
}

// New creates a new inventory subsystem backend.
func New(b kv.KeysPrefixTraversingBucket) *KV {
	return &KV{b: b}
}

// RetrieveInventory queries and returns the inventory values by mapped
// by enrollment ID from the key-value store. Must provide opt and IDs.
func (s *KV) RetrieveInventory(ctx context.Context, opt *storage.SearchOptions) (map[string]storage.Values, error) {
	if opt == nil || len(opt.IDs) < 1 {
		return nil, storage.ErrNoIDs
	}

	r := make(map[string]storage.Values)
	for _, id := range opt.IDs {
		jsonValues, err := s.b.Get(ctx, id)
		if errors.Is(err, kv.ErrKeyNotFound) {
			continue
		} else if err != nil {
			return r, fmt.Errorf("getting values for %s: %w", id, err)
		}

		var values storage.Values
		if err = json.Unmarshal(jsonValues, &values); err != nil {
			return r, fmt.Errorf("unmarshal values for %s: %w", id, err)
		}
		r[id] = values
	}
	return r, nil
}

// StoreInventoryValues stores inventory data about the specified ID.
func (s *KV) StoreInventoryValues(ctx context.Context, id string, newValues storage.Values) error {
	if id == "" {
		return storage.ErrNoIDs
	}
	if len(newValues) == 0 {
		return nil
	}

	jsonValues, err := s.b.Get(ctx, id)
	if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
		return fmt.Errorf("get values: %w", err)
	}

	var values storage.Values
	if len(jsonValues) < 1 {
		values = newValues
	} else {
		// load existing values
		if err = json.Unmarshal(jsonValues, &values); err != nil {
			return fmt.Errorf("unmarshal values: %w", err)
		}

		// merge the new values in
		for k := range newValues {
			values[k] = newValues[k]
		}
	}

	if jsonValues, err = json.Marshal(&values); err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}

	if err = s.b.Set(ctx, id, jsonValues); err != nil {
		return fmt.Errorf("set values: %w", err)
	}

	return nil
}

// DeleteInventory deletes all inventory data for an enrollment ID.
func (s *KV) DeleteInventory(ctx context.Context, id string) error {
	return s.b.Delete(ctx, id)
}
