package mongodb

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/engine/storage/test"
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
		testDatabase = "nanocmd_engine_storage_test"
	}

	ctx := context.Background()
	prefix := "engine_" + strings.NewReplacer("/", "_").Replace(t.Name())
	s, err := New(
		WithURI(testURI),
		WithDatabase(testDatabase),
		WithCollectionPrefix(prefix),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = s.stepCollection.Drop(ctx)
		_ = s.idCmdCollection.Drop(ctx)
		_ = s.eventCollection.Drop(ctx)
		_ = s.statusCollection.Drop(ctx)
		_ = s.client.Disconnect(ctx)
	})

	test.TestEngineStorage(t, func() storage.AllStorage { return s })
}
