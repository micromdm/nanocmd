package workflow

import (
	"errors"
	"fmt"
)

// ErrEventsNotSupported returned from a workflow Event() method.
var ErrEventsNotSupported = errors.New("events not supported for this workflow")

// EventFlag is a bitmask of event types.
type EventFlag uint

// Storage backends (persistent storage) are likely to use these numeric
// values. Treat these as append-only: Order and position matter.
const (
	EventAllCommandResponse EventFlag = 1 << iota
	EventAuthenticate
	EventTokenUpdate
	// TokenUpdate and Enrollment are considered distinct because an
	// enrollment will only enroll once, but TokenUpdates can
	// continually arrive.
	EventEnrollment
	EventCheckOut
	maxEventFlag
)

func (e EventFlag) Valid() bool {
	return e > 0 && e < maxEventFlag
}

func (e EventFlag) String() string {
	switch e {
	case EventAllCommandResponse:
		return "AllCommandResponse"
	case EventAuthenticate:
		return "Authenticate"
	case EventTokenUpdate:
		return "TokenUpdate"
	case EventEnrollment:
		return "Enrollment"
	case EventCheckOut:
		return "CheckOut"
	default:
		return fmt.Sprintf("unknown event type: %d", e)
	}
}

func EventFlagForString(s string) EventFlag {
	switch s {
	case "AllCommandResponse":
		return EventAllCommandResponse
	case "Authenticate":
		return EventAuthenticate
	case "TokenUpdate":
		return EventTokenUpdate
	case "Enrollment":
		return EventEnrollment
	case "CheckOut":
		return EventCheckOut
	default:
		return 0
	}
}

// Event is a specific workflow MDM event.
type Event struct {
	EventFlag
	// EventData is likely a pointer to a struct of the relevent event data.
	// You will need to know the data you're expecting and use Go type
	// conversion to access it if you need it.
	// For example the EventAuthenticate EventFlag will be
	// a `*mdm.Authenticate` under the `interface{}`.
	EventData interface{}
}
