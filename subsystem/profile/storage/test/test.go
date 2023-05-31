package test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
)

func TestProfileStorage(t *testing.T, newStorage func() storage.Storage) {
	s := newStorage()
	ctx := context.Background()

	info := storage.ProfileInfo{Identifier: "com.test", UUID: "01AB"}
	raw := []byte("23CD")

	err := s.StoreProfile(ctx, "test", info, raw)
	if err != nil {
		t.Fatal(err)
	}

	infos, err := s.RetrieveProfileInfos(ctx, []string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	info2, ok := infos["test"]
	if !ok {
		t.Error("key not found after retrieval")
	}

	if !reflect.DeepEqual(info, info2) {
		t.Error("info not equal")
	}

	// test with no names (should return all)
	infos, err = s.RetrieveProfileInfos(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	info2, ok = infos["test"]
	if !ok {
		t.Error("key not found after retrieval (retrieving all keys)")
	}

	if !reflect.DeepEqual(info, info2) {
		t.Error("info not equal")
	}

	raws, err := s.RetrieveRawProfiles(ctx, []string{"test"})
	if err != nil {
		t.Fatal(err)
	}

	raw2, ok := raws["test"]
	if !ok {
		t.Error("key not found after retrieval")
	}

	if !bytes.Equal(raw, raw2) {
		t.Error("raw not equal")
	}

	raws, err = s.RetrieveRawProfiles(ctx, []string{})
	if len(raws) > 0 {
		t.Error("should not return any profiles when using no names")
	}
	if !errors.Is(err, storage.ErrNoNames) {
		t.Fatal("expected ErrNoNames")
	}

	err = s.DeleteProfile(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.RetrieveProfileInfos(ctx, []string{"test"})
	if !errors.Is(err, storage.ErrProfileNotFound) {
		t.Fatal("expected ErrProfileNotFound")
	}

}
