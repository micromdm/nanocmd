package mysql

import (
	"os"
	"testing"

	"github.com/micromdm/nanocmd/engine/storage"
	"github.com/micromdm/nanocmd/engine/storage/test"

	_ "github.com/go-sql-driver/mysql"
)

func TestMySQLStorage(t *testing.T) {
	testDSN := os.Getenv("NANOCMD_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOCMD_MYSQL_STORAGE_TEST_DSN not set")
	}

	s, err := New(WithDSN(testDSN))
	if err != nil {
		t.Fatal(err)
	}

	test.TestEngineStorage(t, func() storage.AllStorage { return s })
}
