// Package devinfolog implements a NanoCMD Workflow that logs information.
package devinfolog

import (
	"context"
	"fmt"
	"strings"

	"github.com/jessepeterson/mdmcommands"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"
	"github.com/micromdm/nanolib/log"
)

const DefaultWorkflowName = "io.micromdm.wf.devinfolog.v1"

// Workflow logs information.
type Workflow struct {
	name   string
	enq    workflow.StepEnqueuer
	ider   uuid.IDer
	logger log.Logger
}

// Options configure [Workflow].
type Option func(*Workflow)

// WithName names the workflow. By default [DefaultWorkflowName] is used.
func WithName(name string) Option {
	return func(w *Workflow) {
		w.name = name
	}
}

// New creates a new [Workflow].
func New(enq workflow.StepEnqueuer, logger log.Logger, opts ...Option) (*Workflow, error) {
	if enq == nil {
		panic("nil enqueuer")
	}
	if logger == nil {
		panic("nil logger")
	}
	w := &Workflow{
		name:   DefaultWorkflowName,
		enq:    enq,
		ider:   uuid.NewUUID(),
		logger: logger,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w, nil
}

// Name returns the name of w.
func (w *Workflow) Name() string {
	return w.name
}

// Config returns nil.
func (w *Workflow) Config() *workflow.Config {
	return nil
}

// NewContextValue returns nil.
func (w *Workflow) NewContextValue(_ string) workflow.ContextMarshaler {
	return nil
}

// Start starts the workflow.
func (w *Workflow) Start(ctx context.Context, step *workflow.StepStart) error {
	// build a DeviceInformation command
	cmd := mdmcommands.NewDeviceInformationCommand(w.ider.ID())
	cmd.Command.Queries = []string{
		"UDID",
		"SerialNumber",
		"Model",
		"ModelName",
		"DeviceName",
		"OSVersion",
		"BuildVersion",
	}

	// assemble our StepEnqueuing
	se := step.NewStepEnqueueing()
	se.Commands = []any{cmd}

	// enqueue our step!
	return w.enq.EnqueueStep(ctx, w, se)
}

func emptyIfNil[T any](in *T) (out T) {
	if in == nil {
		return
	}
	out = *in
	return
}

// StepCompleted is called back whenever any step completes for this workflow.
func (w *Workflow) StepCompleted(ctx context.Context, stepResult *workflow.StepResult) error {
	if len(stepResult.CommandResults) != 1 {
		return workflow.ErrStepResultCommandLenMismatch
	}
	response, ok := stepResult.CommandResults[0].(*mdmcommands.DeviceInformationResponse)
	if !ok {
		return workflow.ErrIncorrectCommandType
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("validating device information response: %w", err)
	}

	// log the query responses
	info := strings.Join([]string{
		"UDID=" + emptyIfNil(response.QueryResponses.UDID),
		"SerialNumber=" + emptyIfNil(response.QueryResponses.SerialNumber),
		"Model=" + emptyIfNil(response.QueryResponses.Model),
		"ModelName=" + emptyIfNil(response.QueryResponses.ModelName),
		"DeviceName=" + emptyIfNil(response.QueryResponses.DeviceName),
		"OSVersion=" + emptyIfNil(response.QueryResponses.OSVersion),
		"BuildVersion=" + emptyIfNil(response.QueryResponses.BuildVersion),
	}, ", ")
	w.logger.Info("msg", info)

	return nil
}

// StepTimeout returns an error; step timeouts are not used in this workflow.
func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

// Event returns an error; events are not supported on this workflow.
func (w *Workflow) Event(ctx context.Context, e *workflow.Event, id string, mdmCtx *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
