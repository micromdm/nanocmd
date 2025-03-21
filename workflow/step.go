package workflow

import (
	"errors"
	"time"
)

var (
	// ErrTimeoutNotUsed returned from a workflow Event() method.
	ErrTimeoutNotUsed = errors.New("workflow does not utilize timeouts")

	// ErrStepResultCommandLenMismatch indicates mismatched MDM commands expected.
	// Steps are enqueued with n MDM commands and should only return with
	// that number of commands. This error is for indicating that this
	// was not the case.
	ErrStepResultCommandLenMismatch = errors.New("mismatched number of commands in step result")

	// ErrUnknownStepName occurs when a workflow encounters a step name
	// it does not know about.
	ErrUnknownStepName = errors.New("unknown step name")

	// ErrIncorrectCommandType occurs when a step's expected command is
	// not of the correct type. Workflows should not depend on the ordering
	// of commands in the returned step command slice.
	ErrIncorrectCommandType = errors.New("incorrect command type")

	// ErrIncorrectContextType indicates a step did not receive the
	// correctly instantiated context type for this step name.
	ErrIncorrectContextType = errors.New("incorrect context type")
)

// StepEnqueueing encapsulates a step and is passed to an enqueuer for command delivery to MDM enrollments.
// Note that a workflow may only enqueue commands to multiple enrollment IDs when starting.
type StepEnqueueing struct {
	StepContext
	IDs      []string // Enrollment IDs
	Commands []interface{}

	// Timeout specifies a timeout. If any of the commands in this step do
	// not complete by this time then the entire step is considered to have
	// timed out.
	Timeout time.Time

	// A step should not be enqueued (that is, sent to enrollments)
	// until after this time has passed. A delay of sorts.
	NotUntil time.Time
}

// StepStart is provided to a workflow when starting a new workflow instance.
// Note that a workflow may only enqueue commands to multiple enrollment IDs when starting.
type StepStart struct {
	StepContext
	Event *Event
	IDs   []string // Enrollment IDs
}

// NewStepEnqueueing preserves some context and IDs from step for enqueueing.
func (step *StepStart) NewStepEnqueueing() *StepEnqueueing {
	if step == nil {
		return nil
	}
	return &StepEnqueueing{
		StepContext: *step.StepContext.NewForEnqueue(),
		IDs:         step.IDs,
	}
}

// StepResult is given to a workflow when a step has completed or timed out.
type StepResult struct {
	StepContext
	ID             string
	CommandResults []interface{}
}

// NewStepEnqueueing preserves some context and IDs from step for enqueueing.
func (step *StepResult) NewStepEnqueueing() *StepEnqueueing {
	if step == nil {
		return nil
	}
	return &StepEnqueueing{
		StepContext: *step.StepContext.NewForEnqueue(),
		IDs:         []string{step.ID},
	}
}
