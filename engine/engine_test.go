package engine

import (
	"context"
	"testing"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/engine/storage/inmem"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

// singleTargetEnqueuer models an MDM server that cannot target one command
// at more than one enrollment (i.e. MicroMDM) by reporting no support for
// multi-targeted commands. It records the enrollment IDs of each enqueueing.
type singleTargetEnqueuer struct {
	enqueuedIDs [][]string
}

func (e *singleTargetEnqueuer) Enqueue(_ context.Context, ids []string, _ []byte) error {
	e.enqueuedIDs = append(e.enqueuedIDs, ids)
	return nil
}

func (e *singleTargetEnqueuer) SupportsMultiCommands() bool { return false }

// oneCommandWorkflow enqueues a single MDM command when started.
type oneCommandWorkflow struct {
	enq  workflow.StepEnqueuer
	ider uuid.IDer
}

func (w *oneCommandWorkflow) Name() string             { return "test.wf.onecommand.v1" }
func (w *oneCommandWorkflow) Config() *workflow.Config { return nil }

func (w *oneCommandWorkflow) NewContextValue(_ string) workflow.ContextMarshaler { return nil }

func (w *oneCommandWorkflow) Start(ctx context.Context, step *workflow.StepStart) error {
	se := step.NewStepEnqueueing()
	se.Commands = []interface{}{mdmcommands.NewDeviceInformationCommand(w.ider.ID())}
	return w.enq.EnqueueStep(ctx, w, se)
}

func (w *oneCommandWorkflow) StepCompleted(_ context.Context, _ *workflow.StepResult) error {
	return nil
}

func (w *oneCommandWorkflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return nil
}

func (w *oneCommandWorkflow) Event(_ context.Context, _ *workflow.Event, _ string, _ *workflow.MDMContext) error {
	return nil
}

// TestStartWorkflowSingleTarget checks that starting a workflow on multiple
// enrollments enqueues to one enrollment at a time when the enqueuer does not
// support multi-targeted commands.
func TestStartWorkflowSingleTarget(t *testing.T) {
	enq := new(singleTargetEnqueuer)
	e := New(inmem.New(), enq)

	w := &oneCommandWorkflow{enq: e, ider: uuid.NewUUID()}
	if err := e.RegisterWorkflow(w); err != nil {
		t.Fatal(err)
	}

	ids := []string{"AAABBBCCC111222333", "DDDEEEFFF444555666"}

	if _, err := e.StartWorkflow(context.Background(), w.Name(), nil, ids, nil, nil); err != nil {
		t.Fatal(err)
	}

	// one enqueueing per enrollment ID; the workflow enqueues one command
	if want, have := len(ids), len(enq.enqueuedIDs); want != have {
		t.Fatalf("enqueue count: want: %d; have: %d", want, have)
	}

	for i, enqueuedIDs := range enq.enqueuedIDs {
		if want, have := 1, len(enqueuedIDs); want != have {
			t.Errorf("enqueue %d: id count: want: %d; have: %d (%v)", i, want, have, enqueuedIDs)
			continue
		}
		if want, have := ids[i], enqueuedIDs[0]; want != have {
			t.Errorf("enqueue %d: id: want: %s; have: %s", i, want, have)
		}
	}
}
