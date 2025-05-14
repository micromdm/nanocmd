package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/micromdm/nanocmd/engine"
	enginestorage "github.com/micromdm/nanocmd/engine/storage/inmem"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage/inmem"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
	"github.com/micromdm/nanocmd/workflow/test"
)

func TestWorkflow(t *testing.T) {
	e := engine.New(enginestorage.New(), &test.NullEnqueuer{})

	c := test.NewCollectingStepEnqueur(e)

	s := inmem.New()

	w, err := New(c, s)
	if err != nil {
		t.Fatal(err)
	}
	w.ider = uuid.NewStaticIDs(
		// note: order is important and depends on values in plist testdata
		"53115671-3f45-49f5-b7cb-22ede8b8afdb",
		"53115671-3f45-49f5-b7cb-22ede8b8afdc",
	)

	ctx := context.Background()

	// enrollment id
	id := "AAABBBCCC111222333"

	e.RegisterWorkflow(w)

	// read it back out of the engine
	w2 := e.Workflow(w.Name()).(*Workflow)

	if w.Name() != w2.Name() {
		t.Fatal("workflow name not equal after registration")
	}

	_, err = e.StartWorkflow(ctx, w.Name(), nil, []string{id}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	steps := c.Steps()

	if want, have := 1, len(steps); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	if want, have := 1, len(steps[0].StepEnqueueing.IDs); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	if want, have := id, steps[0].StepEnqueueing.IDs[0]; want != have {
		t.Errorf("wanted: %s; have: %s", want, have)
	}

	fakeSR := &workflow.StepResult{
		ID:             id,
		CommandResults: []interface{}{"stuff", "stuff2", "stuff3"},
	}
	// pretend our workflow finished, submit some data with the wrong count of results
	err = w.StepCompleted(ctx, fakeSR)
	if !errors.Is(err, workflow.ErrStepResultCommandLenMismatch) {
		t.Error("expected error ErrStepResultCommandLenMismatch")
	}

	err = test.SendCommandEvent(ctx, e, "testdata/devinfo.plist", id, "53115671-3f45-49f5-b7cb-22ede8b8afdb")
	if err != nil {
		t.Fatal(err)
	}

	err = test.SendCommandEvent(ctx, e, "testdata/secinfo.plist", id, "53115671-3f45-49f5-b7cb-22ede8b8afdc")
	if err != nil {
		t.Fatal(err)
	}

	// retreive data from workflow we've processed results
	idValues, err := s.RetrieveInventory(ctx, &storage.SearchOptions{IDs: []string{id}})
	if err != nil {
		t.Error(err)
	}
	if len(idValues) != 1 {
		t.Fatal("enrollment not found")
	}

	values := idValues[id]
	if values == nil {
		t.Fatal("nil inventory")
	}

	if want, have := "MacBookPro11,3", values[storage.KeyModel].(string); want != have {
		t.Errorf("KeyModel: %s; have: %s", want, have)
	}
	if want, have := true, values[storage.KeySIPEnabled].(bool); want != have {
		t.Errorf("KeySIPEnabled: %v; have: %v", want, have)
	}
}
