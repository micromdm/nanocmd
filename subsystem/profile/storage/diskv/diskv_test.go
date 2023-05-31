package diskv

import (
	"os"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/subsystem/profile/storage/test"
)

func TestDiskv(t *testing.T) {
	test.TestProfileStorage(t, func() storage.Storage { return New("teststor") })
	os.RemoveAll("teststor")
}
