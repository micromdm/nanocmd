package diskv

import (
	"os"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage/test"
)

func TestDiskv(t *testing.T) {
	test.TestCMDPlanStorage(t, func() storage.Storage { return New("teststor") })
	os.RemoveAll("teststor")
}
