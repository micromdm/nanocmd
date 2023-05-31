package invprk

import (
	"context"
	"testing"

	"github.com/micromdm/nanocmd/subsystem/inventory/storage/inmem"
)

func TestInvPRK(t *testing.T) {
	ctx := context.Background()
	inv := inmem.New()
	invPRK := NewInvPRK(inv)
	prk := "PRK-321-ZYX"
	err := invPRK.StorePRK(ctx, "ID1", "PRK-321-ZYX")
	if err != nil {
		t.Fatal(err)
	}
	prkRet, err := invPRK.RetrievePRK(ctx, "ID1")
	if err != nil {
		t.Fatal(err)
	}
	if have, want := prk, prkRet; prk != prkRet {
		t.Errorf("have: %v, want: %v", have, want)
	}
}
