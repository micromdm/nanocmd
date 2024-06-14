// Package storage defines types and interfaces to support the inventory subsystem.
package storage

import (
	"context"
	"errors"
)

var (
	ErrNoIDs = errors.New("no ids supplied")
)

// SearchOptions is a basic query for inventory of enrollment IDs.
type SearchOptions struct {
	IDs []string // slice of enrollment IDs to query against
}

// Values maps inventory storage keys to values.
type Values map[string]interface{}

type ReadStorage interface {
	// RetrieveInventory queries and returns the inventory values by mapped by enrollment ID.
	// If no search opt nor IDs are provided an ErrNoIDs should be returned.
	// If IDs are have no inventory data then they should be skipped and
	// omitted from the output with no error.
	RetrieveInventory(ctx context.Context, opt *SearchOptions) (map[string]Values, error)
}

type Storage interface {
	ReadStorage

	// StoreInventoryValues stores inventory data about the specified ID.
	StoreInventoryValues(ctx context.Context, id string, values Values) error

	// DeleteInventory deletes all inventory data for an enrollment ID.
	DeleteInventory(ctx context.Context, id string) error
}
