// Package storage defines types and primitives for workflow engine storage backends.
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/micromdm/nanocmd/workflow"
)

var (
	// ErrInvalidStorageStep is returned when validating storage steps.
	ErrEmptyStorageStep = errors.New("empty storage step")

	// ErrInvalidStepContext is returned when a step context is nil.
	ErrEmptyStepContext = errors.New("invalid step context")

	ErrMissingWorkflowName = errors.New("missing workflow name")
	ErrMissingInstanceID   = errors.New("missing instance id")
	ErrMissingIDs          = errors.New("missing IDs")
	ErrMissingCommands     = errors.New("missing commands")
)

// StepContext is common contextual information for steps.
// An approximately serialized form of a workflow step.
type StepContext struct {
	WorkflowName string // workflow name. used for routing back to the workflow via the engine's step registry.
	InstanceID   string // unique ID of this 'instance' of a workflow.
	Name         string // workflow step name. defined and used by the workflow.
	Context      []byte // workflow step context (in raw marshaled binary form).
}

// Validate checks for missing values.
func (sc *StepContext) Validate() error {
	if sc == nil {
		return ErrEmptyStepContext
	}
	if sc.WorkflowName == "" {
		return ErrMissingWorkflowName
	}
	if sc.InstanceID == "" {
		return ErrMissingInstanceID
	}
	return nil
}

// StepCommandResult is the result of MDM commands from enrollments.
// An approximately serialized form of a workflow step command response.
type StepCommandResult struct {
	CommandUUID  string
	RequestType  string
	ResultReport []byte // raw XML plist result of MDM command
	Completed    bool   // whether this specific command did *not* have a NotNow status
}

var (
	ErrEmptyStepCommandResult = errors.New("empty step command result")
	ErrEmptyResultReport      = errors.New("empty result report")
)

// Validate checks sc for issues.
func (sc *StepCommandResult) Validate() error {
	if sc == nil {
		return ErrEmptyStepCommandResult
	} else if len(sc.ResultReport) < 1 {
		return ErrEmptyResultReport
	}
	return nil
}

// StepCommandRaw is a raw command, its UUID, and request type.
// An approximately serialized form of a workflow step command.
type StepCommandRaw struct {
	CommandUUID string
	RequestType string

	// raw XML plist of MDM command
	// Note that in the case of a step enqueuing a command is considered
	// enqueued with the MDM server if its NotUntil.IsZero() returns
	// true.
	Command []byte
}

// StepEnqueueing is a step for storage that is to be enqueued.
// Ostensibly used to enqueue the commands to MDM and log metadata.
// An approximately serialized form of a workflow step.
type StepEnqueueing struct {
	StepContext
	IDs      []string
	Commands []StepCommandRaw
}

func (se *StepEnqueueing) Validate() error {
	if se == nil {
		return ErrEmptyStorageStep
	}
	if err := se.StepContext.Validate(); err != nil {
		return fmt.Errorf("storage step context invalid: %w", err)
	}
	if len(se.IDs) < 1 {
		return ErrMissingIDs
	}
	if len(se.Commands) < 1 {
		return ErrMissingCommands
	}
	return nil
}

// StepEnqueuingWithConfig is for enqueuing a step with additional configuration.
// An approximately serialized form of a workflow step enqueueing.
type StepEnqueuingWithConfig struct {
	StepEnqueueing

	// wait until after this time to enqueue step commands
	// note that this has implications for the the storage backends.
	// if a NotUntil time is set then the raw commands need to be saved
	// (so that they can be enqueued to the MDM server later). if a
	// NotUntil time is not set then the raw commands can be discarded
	// as we assume that they've been delivered. after a NotUntil time
	// has past then a storage backend can get rid of the raw commands
	// in order to save space.
	NotUntil time.Time

	Timeout time.Time // step times out if not complete by this time
}

func (se *StepEnqueuingWithConfig) Validate() error {
	if se == nil {
		return ErrEmptyStorageStep
	}
	return se.StepEnqueueing.Validate()
}

// StepResult represent the results of all of a step's MDM commands.
// An approximately serialized form of a workflow step result.
type StepResult struct {
	StepContext
	IDs      []string
	Commands []StepCommandResult
}

type WorkflowStatusStorage interface {
	// RetrieveWorkflowStarted returns the last time a workflow was started for id.
	// Returned time should be nil with no error if workflowName has not yet been started for id.
	RetrieveWorkflowStarted(ctx context.Context, id, workflowName string) (time.Time, error)

	// RecordWorkflowStarted stores the started time for workflowName for ids.
	RecordWorkflowStarted(ctx context.Context, ids []string, workflowName string, started time.Time) error

	// ClearWorkflowStatus removes all workflow start times for id.
	ClearWorkflowStatus(ctx context.Context, id string) error
}

// Storage is the primary interface for workflow engine backend storage implementations.
type Storage interface {
	// RetrieveCommandRequestType retrieves a command request type given id and uuid.
	// This effectively tells the engine whether the provided command UUID
	// originated from a workflow enqueueing or not (i.e. whether processing should
	// continue).
	RetrieveCommandRequestType(ctx context.Context, id string, uuid string) (string, bool, error)

	// StoreCommandResponseAndRetrieveCompletedStep stores a command response and returns the completed step for the id.
	// The completed step will be nil if sc does not complete the step.
	// Implementations will need to lookup the step that sc (the Command UUID
	// and Request Type) belongs to. The provided Result Report and Completed
	// values should determine whether this step is completed or not (depending
	// on other pending commands for this id).
	//
	// Any retrieved completed step is assumed to be permanently deleted from storage.
	StoreCommandResponseAndRetrieveCompletedStep(ctx context.Context, id string, sc *StepCommandResult) (*StepResult, error)

	// StoreStep stores a step and its commands for later state tracking.
	// Depending on whether a command was enqueued immediately (NotUntil) the
	// implementation may discard the raw command Plist bytes.
	StoreStep(context.Context, *StepEnqueuingWithConfig, time.Time) error

	// RetrieveOutstandingWorkflowStates finds enrollment IDs with an outstanding workflow step from a given set.
	RetrieveOutstandingWorkflowStatus(ctx context.Context, workflowName string, ids []string) (outstandingIDs []string, err error)

	// CancelSteps cancels workflow steps for id.
	// If workflowName is empty then implementations should cancel all
	// workflow steps for the id. "NotUntil" (future) workflows steps
	// should also be canceled.
	CancelSteps(ctx context.Context, id, workflowName string) error

	WorkflowStatusStorage
}

// WorkerStorage is used by the workflow engine worker for async (scheduled) actions.
type WorkerStorage interface {
	// RetrieveStepsToEnqueue fetches steps to be enqueued that were enqueued "later" with NotUntil.
	// These steps-for-enqueueing will be enqueued to the MDM server for the IDs.
	// Returned steps may not be per-ID and may target mutliple IDs (depending
	// on how the workflow originally enqueued them).
	//
	// Any retrieved step is assumed to be permanently marked as enqueued and
	// will not be retrieved again with this method. As such the raw command
	// bytes can be discarded by the implementation (to e.g. save space).
	RetrieveStepsToEnqueue(ctx context.Context, pushTime time.Time) ([]*StepEnqueueing, error)

	// RetrieveTimedOutSteps fetches steps that have timed out.
	// These steps will be delivered to their workflows as timed out.
	//
	// Any retrieved completed step is assumed to be permanently deleted from storage.
	RetrieveTimedOutSteps(ctx context.Context) ([]*StepResult, error)

	// RetrieveAndMarkRePushed retrieves a set of IDs that need to have APNs re-pushes sent.
	// Marks those IDs as having been pushed to now.
	//
	// Any retrieved IDs are assumed to have neen successfully APNs pushed to and will be marked so at pushTime.
	RetrieveAndMarkRePushed(ctx context.Context, ifBefore time.Time, pushTime time.Time) ([]string, error)
}

type AllStorage interface {
	Storage
	WorkerStorage
	EventSubscriptionStorage
}

// EventSubscription is a user-configured subscription for starting workflows with optional context.
type EventSubscription struct {
	Event        string `json:"event"`
	Workflow     string `json:"workflow"`
	Context      string `json:"context,omitempty"`
	EventContext string `json:"event_context,omitempty"`
}

var (
	ErrEmptyEventSubscription = errors.New("empty event subscription")
	ErrMissingEvent           = errors.New("missing event type")
)

func (es *EventSubscription) Validate() error {
	if es == nil {
		return ErrEmptyEventSubscription
	}
	if es.Event == "" {
		return ErrMissingEvent
	}
	if !workflow.EventFlagForString(es.Event).Valid() {
		return fmt.Errorf("invalid event type: %s", es.Event)
	}
	if es.Workflow == "" {
		return ErrMissingWorkflowName
	}
	return nil
}

// ReadEventSubscriptionStorage describes storage backends that can retrieve and query event subscriptions.
type ReadEventSubscriptionStorage interface {
	RetrieveEventSubscriptions(ctx context.Context, names []string) (map[string]*EventSubscription, error)
	RetrieveEventSubscriptionsByEvent(ctx context.Context, f workflow.EventFlag) ([]*EventSubscription, error)
}

// EventSubscriptionStorage describes storage backends that can also write and delete event subscriptions.
type EventSubscriptionStorage interface {
	ReadEventSubscriptionStorage
	StoreEventSubscription(ctx context.Context, name string, es *EventSubscription) error
	DeleteEventSubscription(ctx context.Context, name string) error
}
