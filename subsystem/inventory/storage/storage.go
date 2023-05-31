// Package storage defines types and interfaces to support the inventory subsystem.
package storage

import (
	"context"
)

// SearchOptions is a basic query for inventory of enrollment IDs.
type SearchOptions struct {
	IDs []string // slice of enrollment IDs to query against
}

// Values maps inventory storage keys to values.
type Values map[string]interface{}

type ReadStorage interface {
	// RetrieveInventory queries and returns the inventory values by mapped by enrollment ID.
	RetrieveInventory(ctx context.Context, opt *SearchOptions) (map[string]Values, error)
}

type Storage interface {
	ReadStorage
	StoreInventoryValues(ctx context.Context, id string, values Values) error
	DeleteInventory(ctx context.Context, id string) error
}
