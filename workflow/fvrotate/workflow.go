// Package fvrotate implements a NanoCMD Workflow for FileVault key rotation.
package fvrotate

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanocmd/logkeys"
	"github.com/micromdm/nanocmd/subsystem/filevault/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

const WorkflowName = "io.micromdm.wf.fvrotate.v1"

type Workflow struct {
	enq    workflow.StepEnqueuer
	ider   uuid.IDer
	logger log.Logger
	store  storage.FVRotate
}

type Option func(*Workflow)

func WithLogger(logger log.Logger) Option {
	return func(w *Workflow) {
		w.logger = logger
	}
}

func New(q workflow.StepEnqueuer, store storage.FVRotate, opts ...Option) (*Workflow, error) {
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

func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	for _, id := range step.IDs {
		// fetch cert & PRK
		certRaw, err := w.store.RetrievePRKCertRaw(ctx, id)
		if err != nil {
			return fmt.Errorf("retrieving PRK cert raw for %s: %w", id, err)
		}
		prk, err := w.store.RetrievePRK(ctx, id)
		if err != nil {
			return fmt.Errorf("retrieving PRK for %s: %w", id, err)
		}

		// create MDM command
		cmd := mdmcommands.NewRotateFileVaultKeyCommand(w.ider.ID())
		cmd.Command.KeyType = "personal"
		cmd.Command.FileVaultUnlock.Password = &prk
		cmd.Command.ReplyEncryptionCertificate = &certRaw

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
	response, ok := stepResult.CommandResults[0].(*mdmcommands.RotateFileVaultKeyResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("validating rotate response: %w", err)
	}

	if response.RotateResult == nil || response.RotateResult.EncryptedNewRecoveryKey == nil {
		return errors.New("rotate result has missing (nil) fields")
	}

	if err := w.store.EscrowPRK(ctx, stepResult.ID, *response.RotateResult.EncryptedNewRecoveryKey); err != nil {
		return fmt.Errorf("escrow PRK for %s: %w", stepResult.ID, err)
	}

	ctxlog.Logger(ctx, w.logger).Debug(
		logkeys.InstanceID, stepResult.InstanceID,
		logkeys.EnrollmentID, stepResult.ID,
		logkeys.Message, "escrowed PRK",
	)
	return nil
}

func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

func (w *Workflow) Event(_ context.Context, _ *workflow.Event, _ string, _ *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
