// Package invprk implements retrieving and storing PRKs in inventory storage.
package invprk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
)

// InvPRK retrieves and stores PRKs in inventory storage.
type InvPRK struct {
	i storage.Storage
}

func NewInvPRK(i storage.Storage) *InvPRK {
	return &InvPRK{i: i}
}

func (s *InvPRK) StorePRK(ctx context.Context, id, prk string) error {
	return s.i.StoreInventoryValues(ctx, id, storage.Values{
		storage.KeyLastSource: "InvPRK",
		storage.KeyModified:   time.Now(),
		storage.KeyPRK:        prk,
	})
}

func (s *InvPRK) RetrievePRK(ctx context.Context, id string) (string, error) {
	idVals, err := s.i.RetrieveInventory(ctx, &storage.SearchOptions{IDs: []string{id}})
	if err != nil {
		return "", fmt.Errorf("retrieve inventory: %w", err)
	}
	if idVals == nil {
		return "", errors.New("no values returned from inventory")
	}
	vals, ok := idVals[id]
	if !ok || vals == nil {
		return "", fmt.Errorf("id not in inventory: %s", id)
	}
	prkVal, ok := vals[storage.KeyPRK]
	var prk string
	if !ok || prkVal == nil {
		return "", fmt.Errorf("inventory does not contain PRK value: %s", id)
	} else if prk, ok = prkVal.(string); !ok {
		return "", errors.New("PRK incorrect inventory type")
	}
	return prk, nil
}
