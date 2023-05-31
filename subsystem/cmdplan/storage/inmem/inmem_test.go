package inmem

import (
	"testing"

	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage/test"
)

func TestInMem(t *testing.T) {
	test.TestCMDPlanStorage(t, func() storage.Storage { return New() })
}
