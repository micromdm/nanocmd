package inventory

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/micromdm/nanocmd/engine"
	enginestorage "github.com/micromdm/nanocmd/engine/storage/inmem"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage/inmem"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

type nullEnqueuer struct{}

func (n *nullEnqueuer) Enqueue(_ context.Context, _ []string, _ []byte) error { return nil }

func (n *nullEnqueuer) SupportsMultiCommands() bool { return true }

type testStep struct {
	wfName string
	es     *workflow.StepEnqueueing
}

type collectionEnqueuer struct {
	next  workflow.StepEnqueuer
	steps []testStep
}

func (c *collectionEnqueuer) EnqueueStep(ctx context.Context, n workflow.Namer, es *workflow.StepEnqueueing) error {
	c.steps = append(c.steps, testStep{
		wfName: n.Name(),
		es:     es,
	})
	return c.next.EnqueueStep(ctx, n, es)
}

func TestWorkflow(t *testing.T) {
	e := engine.New(enginestorage.New(), &nullEnqueuer{})

	c := &collectionEnqueuer{next: e}

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
		t.Fatal("workflows not equal after registration")
	}

	_, err = e.StartWorkflow(ctx, w.Name(), nil, []string{id}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if want, have := 1, len(c.steps); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	if want, have := 1, len(c.steps[0].es.IDs); want != have {
		t.Fatalf("wanted: %d; have: %d", want, have)
	}

	if want, have := id, c.steps[0].es.IDs[0]; want != have {
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

	model := "MacBookPro11,3"

	devInfo, err := os.ReadFile("testdata/devinfo.plist")
	if err != nil {
		t.Fatal(err)
	}

	err = e.MDMCommandResponseEvent(ctx, id, "53115671-3f45-49f5-b7cb-22ede8b8afdb", devInfo, nil)
	if err != nil {
		t.Fatal(err)
	}

	secInfo, err := os.ReadFile("testdata/secinfo.plist")
	if err != nil {
		t.Fatal(err)
	}

	err = e.MDMCommandResponseEvent(ctx, id, "53115671-3f45-49f5-b7cb-22ede8b8afdc", secInfo, nil)
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

	if wanted, have := model, values[storage.KeyModel]; wanted != have {
		t.Errorf("wanted: %s; have: %s", wanted, have)
	}
	if wanted, have := true, values[storage.KeySIPEnabled]; wanted != have {
		t.Errorf("wanted: %v; have: %v", wanted, have)
	}
}
