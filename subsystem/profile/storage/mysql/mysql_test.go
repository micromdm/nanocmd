package mysql

import (
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/subsystem/profile/storage/test"
)

func TestMySQLStorage(t *testing.T) {
	testDSN := os.Getenv("NANODEP_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANODEP_MYSQL_STORAGE_TEST_DSN not set")
	}

	test.TestProfileStorage(t, func() (storage.Storage, error) {
		return New(WithDSN(testDSN))
	})
}
