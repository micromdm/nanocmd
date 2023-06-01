// Pacakge lock implements a DeviceLock PIN escrow workflow.
package lock

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/ctxlog"
	"github.com/micromdm/nanocmd/log/logkeys"
	"github.com/micromdm/nanocmd/subsystem/inventory/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

const WorkflowName = "io.micromdm.wf.lock.v1"

type Workflow struct {
	enq    workflow.StepEnqueuer
	ider   uuid.IDer
	logger log.Logger
	store  storage.Storage
}

type Option func(*Workflow)

func WithLogger(logger log.Logger) Option {
	return func(w *Workflow) {
		w.logger = logger
	}
}

func New(q workflow.StepEnqueuer, store storage.Storage, opts ...Option) (*Workflow, error) {
	w := &Workflow{
		enq:    q,
		ider:   uuid.NewUUID(),
		logger: log.NopLogger,
		store:  store,
	}
	for _, opt := range opts {
		opt(w)
	}
	w.logger = w.logger.With(logkeys.WorkflowName, w.Name())
	return w, nil
}

func (w *Workflow) Name() string {
	return WorkflowName
}

func (w *Workflow) Config() *workflow.Config {
	return nil
}

func (w *Workflow) NewContextValue(name string) workflow.ContextMarshaler {
	return nil
}

func randomDigits(n int) string {
	digits := make([]byte, n)
	for i := 0; i < n; i++ {
		digits[i] = byte(rand.Intn(10) + '0')
	}
	return string(digits)
}

func (w *Workflow) storeLock(ctx context.Context, id, pin string) error {
	return w.store.StoreInventoryValues(ctx, id, storage.Values{
		WorkflowName + ".pin":  pin,
		WorkflowName + ".sent": time.Now(),
		storage.KeyLastSource:  WorkflowName,
	})
}

func (w *Workflow) updateLock(ctx context.Context, id, msg string) error {
	v := storage.Values{
		WorkflowName + ".received": time.Now(),
		storage.KeyLastSource:      WorkflowName,
	}
	if msg != "" {
		v[WorkflowName+".message_result"] = msg
	}
	return w.store.StoreInventoryValues(ctx, id, v)
}

func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	for _, id := range step.IDs {
		// generate us a PIN
		pin := randomDigits(6)

		err := w.storeLock(ctx, id, pin)
		if err != nil {
			return fmt.Errorf("store inventory values for %s: %w", id, err)
		}

		// create MDM command
		cmd := mdmcommands.NewDeviceLockCommand(w.ider.ID())
		cmd.Command.PIN = &pin

		// assemble our StepEnqueuing
		se := step.NewStepEnqueueing()
		se.IDs = []string{id} // scope to just this ID we're iterating over
		se.Commands = []interface{}{cmd}

		// enqueue our step!
		if err = w.enq.EnqueueStep(ctx, w, se); err != nil {
			return fmt.Errorf("enqueueing step for %s: %w", id, err)
		}
	}
	return nil
}

func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	response, ok := stepResult.CommandResults[0].(*mdmcommands.DeviceLockResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("validating lock response: %w", err)
	}

	logs := []interface{}{
		logkeys.InstanceID, stepResult.InstanceID,
		logkeys.EnrollmentID, stepResult.ID,
		logkeys.Message, "lock received",
	}
	msg := ""
	if response.MessageResult != nil && *response.MessageResult != "" {
		msg = *response.MessageResult
		logs = append(logs, "message_result", msg)
	}
	ctxlog.Logger(ctx, w.logger).Debug(logs...)

	if err := w.updateLock(ctx, stepResult.ID, msg); err != nil {
		return fmt.Errorf("update inventory values for %s: %w", stepResult.ID, err)
	}

	return nil
}

func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

func (w *Workflow) Event(_ context.Context, _ *workflow.Event, _ string, _ *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
