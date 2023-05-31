package workflow

import (
	"encoding"
	"errors"
	"strconv"
)

// ContextMarshaler marshals and unmarshals types to and from byte slices.
// This encapsulates arbitrary context types to be passed around and
// stored as binary blobs by components (that are not a workflow ) that
// don't need to care about what the contents are (e.g. storage backends
// or HTTP handlers).
type ContextMarshaler interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// MDMContext contains context related to the MDM server, enrollment, and/or MDM request.
type MDMContext struct {
	// Params are the URL parameters included in the MDM request from an
	// enrollment. These parameters would be set on the `CheckInURL` or
	// `ServerURL` parameters in the enrollment profile. Note because
	// these come from a connecting MDM client they may not be present
	// in all contexts — only those that originate from an MDM request.
	Params map[string]string
}

// StepContext contains context for a step.
type StepContext struct {
	// MDM client/server context. Note that a step can be more than one
	// MDM command response. This means MDMContext will likely only
	// come from the very last command to be seen that completed
	// the step. Previous MDMConext will not be seen/provided.
	MDMContext

	InstanceID string // Unique identifier of the workflow instance

	// Name is used by the workflow to identify which step is being processed.
	// This value can help the workflow differentiate steps for a multi-step workflow.
	// It is also passed to the NewContext() method to determine the data
	// type for unmarshalling. Name is empty when starting a workflow.
	Name string

	// Context is a generic holder of data that workflows will be handed when processing steps.
	// Usually this will be an instance of whatever value the workflow
	// NewContext() method returns for a given step name.
	Context ContextMarshaler
}

// NewForEnqueue is a helper for creating a new context from c.
// It copies the instance ID — ostensibly for creating a new Context for
// the next step enqueueing.
func (c *StepContext) NewForEnqueue() *StepContext {
	return &StepContext{InstanceID: c.InstanceID}
}

// StringContext is a simple string ContextMarshaler.
type StringContext string

// MarshalBinary converts c into a byte slice.
func (c *StringContext) MarshalBinary() ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil value")
	}
	return []byte(*c), nil
}

// UnmarshalBinary converts and loads data into c.
func (c *StringContext) UnmarshalBinary(data []byte) error {
	if c == nil {
		return errors.New("nil value")
	}
	*c = StringContext(data)
	return nil
}

// IntContext is a simple integer ContextMarshaler.
type IntContext int

// MarshalBinary converts c into a byte slice.
func (c *IntContext) MarshalBinary() ([]byte, error) {
	if c == nil {
		return nil, errors.New("nil value")
	}
	return []byte(strconv.Itoa(int(*c))), nil
}

// UnmarshalBinary converts and loads data into c.
func (c *IntContext) UnmarshalBinary(data []byte) error {
	if c == nil {
		return errors.New("nil value")
	}
	i, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	*c = IntContext(i)
	return nil
}
