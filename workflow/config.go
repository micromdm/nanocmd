package workflow

import (
	"time"
)

// Exclusivity is the exclusivity "mode" for a workflow.
type Exclusivity uint

const (
	// Workflow can only run if no other pending step for this workflow
	// for an enrollment id exists in the system.
	// This is the default mode (0 value).
	Exclusive Exclusivity = iota

	// Workflow can run simultaneous instances for an enrollment ID.
	MultipleSimultaneous

	maxExclusivity
)

func (we Exclusivity) Valid() bool {
	return we < maxExclusivity
}

// Config represents static workflow-wide configuration.
type Config struct {
	// workflow default step timeout.
	// if a workflow does not specify a timeout when enqueueing steps
	// then this default is used. if this default is not sepcified then
	// the engine's default timeout is used.
	Timeout time.Duration

	// defines the workflow exclusivity style
	Exclusivity

	// workflows have the option to receive command responses from
	// any MDM command request type that the engine enqueues (i.e. from
	// other workflows) — not just the commands that this workflow
	// enqueues. specifiy the Request Types for those command here. They
	// will be received by the workflow as an event.
	AllCommandResponseRequestTypes []string

	// event subscriptions. this workflow will get called every time
	// these events happen. use bitwise OR to specify multiple events.
	Events EventFlag
}
