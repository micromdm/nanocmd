// Package certprof implements a NanoCMD Workflow installs a profile based on
// a certificate list command response.
package certprof

import (
	"context"

	"github.com/micromdm/nanocmd/subsystem/profile/storage"
	"github.com/micromdm/nanocmd/utils/uuid"
	"github.com/micromdm/nanocmd/workflow"

	"github.com/micromdm/nanolib/log"
)

const (
	DefaultWorkflowName = "io.micromdm.wf.certprof.v1"

	stepNameProfile = "profile"
)

// Workflow is a workflow that conditionally installs profiles based on the list of certificates.
type Workflow struct {
	name   string
	enq    workflow.StepEnqueuer
	ider   uuid.IDer
	store  storage.ReadRawStorage
	logger log.Logger
}

type Option func(*Workflow) error

// WithLogger configures logger on the workflow.
func WithLogger(logger log.Logger) Option {
	return func(w *Workflow) error {
		w.logger = logger
		return nil
	}
}

// WithName sets the workflow name. If not set a default will be used.
// This can be useful to separate an "exclusivity domain" for the same workflow.
func WithName(name string) Option {
	return func(w *Workflow) error {
		w.name = name
		return nil
	}
}

func New(enq workflow.StepEnqueuer, store storage.ReadRawStorage, opts ...Option) (*Workflow, error) {
	if enq == nil {
		panic("nil enqueuer")
	}
	if store == nil {
		panic("nil store")
	}
	w := &Workflow{
		name:   DefaultWorkflowName,
		enq:    enq,
		ider:   uuid.NewUUID(),
		store:  store,
		logger: log.NopLogger,
	}
	for _, opt := range opts {
		if err := opt(w); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// Name returns the workflow name.
func (w *Workflow) Name() string {
	return w.name
}

// Config returns nil. This workflow does not specify a workflow Conifg.
func (w *Workflow) Config() *workflow.Config {
	return nil
}

// NewContextValue returns a new [Context] regardless of input.
func (w *Workflow) NewContextValue(_ string) workflow.ContextMarshaler {
	return new(Context)
}

// StepTimeout is a stub handler for the workflow interface.
// This workflow does not support step timeout handling.
func (w *Workflow) StepTimeout(_ context.Context, _ *workflow.StepResult) error {
	return workflow.ErrTimeoutNotUsed
}

// Event is a stub handler for the workflow interface.
// This workflow does not support events.
func (w *Workflow) Event(ctx context.Context, e *workflow.Event, id string, mdmCtx *workflow.MDMContext) error {
	return workflow.ErrEventsNotSupported
}
