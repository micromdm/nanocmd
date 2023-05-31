// Package diskv implements a diskv-backed inventory subsystem storage backend.
package diskv

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is an on-disk enrollment inventory data store.
type Diskv struct {
	diskv *diskv.Diskv
}

// New creates a new initialized inventory data store.
func New(path string) *Diskv {
	flatTransform := func(s string) []string { return []string{} }
	return &Diskv{
		diskv: diskv.New(diskv.Options{
			BasePath:     filepath.Join(path, "inventory"),
			Transform:    flatTransform,
			CacheSizeMax: 1024 * 1024,
		}),
	}
}

// RetrieveInventory retrieves the inventory data for enrollment IDs.
func (s *Diskv) RetrieveInventory(ctx context.Context, opt *storage.SearchOptions) (map[string]storage.Values, error) {
	ret := make(map[string]storage.Values)
	for _, id := range opt.IDs {
		if !s.diskv.Has(id) {
			continue
		}
		raw, err := s.diskv.Read(id)
		if err != nil {
			return ret, fmt.Errorf("reading values for %s: %w", id, err)
		}
		var vals storage.Values
		if err = json.Unmarshal(raw, &vals); err != nil {
			return ret, fmt.Errorf("unmarshal values for %s: %w", id, err)
		}
		ret[id] = vals
	}
	return ret, nil
}

// StoreInventoryValues stores inventory data about the specified ID.
func (s *Diskv) StoreInventoryValues(ctx context.Context, id string, values storage.Values) error {
	var err error
	var raw []byte
	var vals storage.Values
	if s.diskv.Has(id) {
		// this is likely race-prone as we perform a read-process-write on the same key.
		if raw, err = s.diskv.Read(id); err != nil {
			return fmt.Errorf("reading values: %w", err)
		}
		if len(raw) > 0 {
			if err = json.Unmarshal(raw, &vals); err != nil {
				return fmt.Errorf("unmarshal values: %w", err)
			}
			if vals != nil {
				for k := range values {
					vals[k] = values[k]
				}
			}
		}
	}
	if vals == nil {
		vals = values
	}
	if raw, err = json.Marshal(vals); err != nil {
		return fmt.Errorf("marshal values: %w", err)
	}
	if err = s.diskv.Write(id, raw); err != nil {
		return fmt.Errorf("write values: %w", err)
	}
	return nil
}

// DeleteInventory deletes all inventory data for an enrollment ID.
func (s *Diskv) DeleteInventory(ctx context.Context, id string) error {
	return s.diskv.Erase(id)
}
