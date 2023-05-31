package inmem

import (
	"testing"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage/test"
)

func TestInMem(t *testing.T) {
	test.TestStorage(t, func() storage.Storage { return New() })
}
