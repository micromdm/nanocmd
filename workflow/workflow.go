package workflow

import "context"

// Namers provide a name string.
type Namer interface {
	// Name returns the name of the workflow; reverse-DNS style by convention.
	// This string is generally used to route actions to this workflow.
	Name() string
}

// Workflows send MDM commands and process the results using steps.
type Workflow interface {
	Namer

	// Config returns the workflow configuration.
	Config() *Config

	// NewContextValue returns a newly instantiated context value.
	// This will usally be used by a workflow engine to unmarshal and pass in
	// stored context on a StepContext.
	NewContextValue(stepName string) ContextMarshaler

	// Start starts a new workflow instance for MDM enrollments.
	Start(context.Context, *StepStart) error

	// StepCompleted is the action called when all step MDM commands have reported results.
	// Note that these results may be errors, but NotNow responses are handled for the workflow.
	StepCompleted(context.Context, *StepResult) error

	// StepTimeout occurs when at least one command in a step has failed to complete in time.
	// Timeouts are defined by the step, then any workflow default, then
	// any engine default.
	StepTimeout(context.Context, *StepResult) error

	// Event is called when MDM events happen that are intended for this workflow.
	// A workflow can subscribe to events in its Config struct.
	Event(ctx context.Context, e *Event, id string, mdmCtx *MDMContext) error
}

// StepEnqueuers send steps (MDM commands) to enrollments.
type StepEnqueuer interface {
	// EnqueueStep enqueues MDM commands to ids in StepEnqueue.
	// The enqueing system should be able to find this workflow again with Namer.
	EnqueueStep(context.Context, Namer, *StepEnqueueing) error
}
