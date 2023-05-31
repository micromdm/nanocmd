package test

import (
	"context"
	"reflect"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/cmdplan/storage"
)

func TestCMDPlanStorage(t *testing.T, newStorage func() storage.Storage) {
	s := newStorage()
	ctx := context.Background()

	plan := &storage.CMDPlan{
		ProfileNames: []string{"hello"},
		ManifestURLs: []string{"gopher://example.com/1/news"},
	}

	err := s.StoreCMDPlan(ctx, "test1", plan)
	if err != nil {
		t.Fatal(err)
	}

	plan2, err := s.RetrieveCMDPlan(ctx, "test1")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(plan, plan2) {
		t.Error("not equal")
	}

	err = s.DeleteCMDPlan(ctx, "test1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.RetrieveCMDPlan(ctx, "test1")
	if err == nil {
		t.Fatal("expected error")
	}
}
