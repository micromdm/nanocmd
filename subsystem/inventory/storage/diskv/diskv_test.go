package diskv

import (
	"os"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage/test"
)

func TestDiskv(t *testing.T) {
	test.TestStorage(t, func() storage.Storage { return New("teststor") })
	os.RemoveAll("teststor")
}
