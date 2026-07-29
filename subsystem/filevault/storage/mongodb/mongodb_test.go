package mongodb

import (
	"bytes"
	"context"
	"crypto/x509"
	"os"
	"strings"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/filevault/storage/invprk"
	invstorage "github.com/micromdm/nanocmd/subsystem/inventory/storage/inmem"
	"github.com/smallstep/pkcs7"
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
		testDatabase = "nanocmd_filevault_storage_test"
	}

	ctx := context.Background()
	testCollection := "subsystem_filevault_" + strings.NewReplacer("/", "_").Replace(t.Name())
	prkStorage := invprk.NewInvPRK(invstorage.New())
	s, err := New(
		prkStorage,
		WithURI(testURI),
		WithDatabase(testDatabase),
		WithCollectionName(testCollection),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = s.collection.Drop(ctx)
		_ = s.client.Disconnect(ctx)
	})

	certRaw, err := s.RetrievePRKCertRaw(ctx, "ID1")
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(certRaw)
	if err != nil {
		t.Fatal(err)
	}

	s2, err := New(prkStorage, WithCollection(s.collection))
	if err != nil {
		t.Fatal(err)
	}
	certRaw2, err := s2.RetrievePRKCertRaw(ctx, "ID1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(certRaw, certRaw2) {
		t.Fatal("expected persisted FileVault certificate")
	}

	prk := "PRK-321-ZYX"
	cms, err := pkcs7.Encrypt([]byte(prk), []*x509.Certificate{cert})
	if err != nil {
		t.Fatal(err)
	}

	if err = s.EscrowPRK(ctx, "ID1", cms); err != nil {
		t.Fatal(err)
	}

	prkRet, err := s.RetrievePRK(ctx, "ID1")
	if err != nil {
		t.Fatal(err)
	}
	if have, want := prkRet, prk; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}
}
