// Package fvenable implements a NanoCMD Workflow for enabling FileVault on a Mac.
package fvenable

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/log"
	"github.com/micromdm/nanocmd/log/ctxlog"
	"github.com/micromdm/nanocmd/log/logkeys"
	fvstorage "github.com/micromdm/nanocmd/subsystem/filevault/storage"
	profstorage "github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
)

const WorkflowName = "io.micromdm.wf.fvenable.v1"

type Workflow struct {
	enq       workflow.StepEnqueuer
	ider      uuid.IDer
	logger    log.Logger
	store     fvstorage.FVEnable
	profStore profstorage.ReadStorage
}

const (
	stepNameInstall = "install"
	stepNamePoll    = "poll"

	pollInterval = 2 * time.Minute // polling interval
	pollCounter  = 180             // how many times we poll (total ~6 hrs)
)

type Option func(*Workflow)

func WithLogger(logger log.Logger) Option {
	return func(w *Workflow) {
		w.logger = logger
	}
}

func New(enq workflow.StepEnqueuer, store fvstorage.FVEnable, profStore profstorage.ReadStorage, opts ...Option) (*Workflow, error) {
	if store == nil {
		return nil, errors.New("empty store")
	}
	w := &Workflow{
		enq:       enq,
		ider:      uuid.NewUUID(),
		logger:    log.NopLogger,
		store:     store,
		profStore: profStore,
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
	if name == stepNamePoll {
		return new(workflow.IntContext) // poll counter
	}
	return nil
}

// profileTemplate retrieves the FileVault profile template from the
// profile store or falls back to the static/hardcoded profile.
func (w *Workflow) profileTemplate(ctx context.Context) ([]byte, error) {
	profiles, err := w.profStore.RetrieveRawProfiles(ctx, []string{WorkflowName})
	if err != nil {
		return nil, fmt.Errorf("retrieving profile %s: %w", WorkflowName, err)
	}
	var profile []byte
	if profiles != nil {
		profile = profiles[WorkflowName]
	}
	if len(profile) < 1 {
		profile = []byte(ProfileTemplate)
	}
	return profile, nil
}

// createInstallProfileCommand creates the InstallProfile command for FV profiles.
// It will inject the certificate used for encrypting the PRK into the
// profile using text replacement.
func (w *Workflow) createInstallProfileCommand(ctx context.Context, id string, profileTemplate []byte) (*mdmcommands.InstallProfileCommand, error) {
	// retrieve the encryption certificate
	certRaw, err := w.store.RetrievePRKCertRaw(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("retrieving PRK cert raw: %w", err)
	}
	// profiles encode binary data as base64.
	certB64 := []byte(base64.StdEncoding.EncodeToString(certRaw))
	// inject the certificate into the profile payload.
	profile := bytes.Replace(profileTemplate, []byte("__CERTIFICATE__"), certB64, 1)
	// generate the command
	cmd := mdmcommands.NewInstallProfileCommand(w.ider.ID())
	cmd.Command.Payload = profile
	return cmd, nil
}

// Start installs the initial FileVault profile.
func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	profTemplate, err := w.profileTemplate(ctx)
	if err != nil {
		return fmt.Errorf("getting profile: %w", err)
	}

	for _, id := range step.IDs {
		cmd, err := w.createInstallProfileCommand(ctx, id, profTemplate)
		if err != nil {
			return fmt.Errorf("creating install profile command for %s: %w", id, err)
		}

		// assemble our StepEnqueuing
		se := step.NewStepEnqueueing()
		se.IDs = []string{id} // scope to just this ID we're iterating over
		se.Commands = []interface{}{cmd}
		se.Name = stepNameInstall

		// enqueue our step!
		if err = w.enq.EnqueueStep(ctx, w, se); err != nil {
			return fmt.Errorf("enqueueing step for %s: %w", id, err)
		}
	}
	return err
}

// installStepCompleted verifies a good profile install and initiates polling with the SecurityInfo command.
func (w *Workflow) installStepCompleted(ctx context.Context, logger log.Logger, stepResult *workflow.StepResult) error {
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	response, ok := stepResult.CommandResults[0].(*mdmcommands.InstallProfileResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("validating install response: %w", err)
	}

	logger.Debug(logkeys.Message, "install completed, initiating polling")

	// assemble our StepEnqueuing to kick off our polling
	se := stepResult.NewStepEnqueueing()
	se.Commands = []interface{}{mdmcommands.NewSecurityInfoCommand(w.ider.ID())}
	se.Name = stepNamePoll
	ctxVal := workflow.IntContext(pollCounter)
	se.Context = &ctxVal
	se.NotUntil = time.Now().Add(pollInterval)

	// enqueue our step!
	return w.enq.EnqueueStep(ctx, w, se)
}

func boolPtr(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func (w *Workflow) pollStepCompleted(ctx context.Context, logger log.Logger, stepResult *workflow.StepResult) error {
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	response, ok := stepResult.CommandResults[0].(*mdmcommands.SecurityInfoResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}

	ctxVal, ok := stepResult.Context.(*workflow.IntContext)
	if !ok {
		return errors.New("invalid context value type")
	}
	if *ctxVal < 0 {
		return errors.New("maximum poll counter reached, ending workflow")
	}

	if err := response.Validate(); err != nil {
		logger.Info(
			logkeys.Message, "validating poll response",
			logkeys.Error, err,
		)
	} else if !boolPtr(response.SecurityInfo.FDEEnabled) {
		logger.Info(
			logkeys.Message, "checking FDE enabled",
			logkeys.Error, "FDE not enabled",
		)
	} else if response.SecurityInfo.FDEPersonalRecoveryKeyCMS == nil {
		logger.Info(
			logkeys.Message, "checking PRK CMS",
			logkeys.Error, "FDE enabled but PRK CMS not present",
		)
	} else if err = w.store.EscrowPRK(ctx, stepResult.ID, *response.SecurityInfo.FDEPersonalRecoveryKeyCMS); err != nil {
		logger.Info(
			logkeys.Message, "escrow PRK",
			logkeys.Error, err,
		)
	} else {
		logger.Debug(
			logkeys.Message, "escrowed PRK",
		)
		return nil
	}

	*ctxVal -= 1 // subtract one from our poll counter
	logger.Debug(
		logkeys.Message, "continuing polling",
		"count_remaining", int(*ctxVal),
	)

	// assemble our StepEnqueuing
	se := stepResult.NewStepEnqueueing()
	se.Commands = []interface{}{mdmcommands.NewSecurityInfoCommand(w.ider.ID())}
	se.Name = stepNamePoll
	se.NotUntil = (time.Now().Add(pollInterval))
	se.Context = ctxVal

	// enqueue our step!
	return w.enq.EnqueueStep(ctx, w, se)
}

func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	logger := ctxlog.Logger(ctx, w.logger).With(
		logkeys.InstanceID, stepResult.InstanceID,
		logkeys.EnrollmentID, stepResult.ID,
	)
	switch stepResult.Name {
	case stepNameInstall:
		return w.installStepCompleted(ctx, logger, stepResult)
	case stepNamePoll:
		return w.pollStepCompleted(ctx, logger, stepResult)
	default:
		return fmt.Errorf("%w: %s", workflow.ErrUnknownStepName, stepResult.Name)
	}
}

func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

func (w *Workflow) Event(_ context.Context, _ *workflow.Event, _ string, _ *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
