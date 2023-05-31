package diskv

import (
	"os"
	"testing"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/engine/storage/test"
)

func TestDiskvStorage(t *testing.T) {
	test.TestEngineStorage(t, func() storage.AllStorage { return New("teststor") })
	os.RemoveAll("teststor")
}
