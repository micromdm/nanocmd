package mongodb

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage/test"
)

func TestMongoDBStorage(t *testing.T) {
	testURI := os.Getenv("NANOCMD_MONGODB_STORAGE_TEST_URI")
	if testURI == "" {
		testURI = os.Getenv("NANODEP_MONGODB_STORAGE_TEST_URI")
	}
	if testURI == "" {
		t.Skip("NANOCMD_MONGODB_STORAGE_TEST_URI not set")
	}

	testDatabase := os.Getenv("NANOCMD_MONGODB_STORAGE_TEST_DATABASE")
	if testDatabase == "" {
		testDatabase = "nanocmd_inventory_storage_test"
	}

	testCollection := "subsystem_inventory_" + strings.NewReplacer("/", "_").Replace(t.Name())
	s, err := New(
		WithURI(testURI),
		WithDatabase(testDatabase),
		WithCollectionName(testCollection),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := s.collection.Drop(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = s.collection.Drop(ctx)
		_ = s.client.Disconnect(ctx)
	})

	test.TestStorage(t, func() storage.Storage { return s })
}
