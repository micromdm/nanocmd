package test

import (
	"context"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
)

func TestStorage(t *testing.T, newStorage func() storage.Storage) {
	s := newStorage()
	ctx := context.Background()

	id := "AA11BB22"

	updValues := storage.Values{"a": "hi"}

	err := s.StoreInventoryValues(ctx, id, updValues)
	if err != nil {
		t.Error(err)
	}

	q := &storage.SearchOptions{IDs: []string{id}}
	idVals, err := s.RetrieveInventory(ctx, q)
	if err != nil {
		t.Error(err)
	}

	vals, ok := idVals[id]
	if !ok {
		t.Error("expected id in id values map")
	}

	testVal, ok := vals["a"]
	if !ok {
		t.Error("expected map key exists")
	} else {
		testValString, ok := testVal.(string)
		if !ok {
			t.Error("test value incorrect")
		}

		if have, want := testValString, updValues["a"]; have != want {
			t.Errorf("want: %v, have: %v", want, have)
		}
	}

	err = s.DeleteInventory(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	idVals, err = s.RetrieveInventory(ctx, q)
	if err != nil {
		t.Error(err)
	}

	_, ok = idVals[id]
	if ok {
		t.Error("expected id to be missing in id values map")
	}
}
