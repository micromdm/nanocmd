// Package logkeys defines some static logging keys for consistent structured logging output.
// Mostly exists as a mental aid when drafting log messages.
package logkeys

const (
	Message = "msg"
	Error   = "err"

	// an MDM enrollment ID. i.e. a UDID, EnrollmentID, etc.
	EnrollmentID = "id"

	// in cases where we might need to log multiple enrollment IDs but only
	// want to log the first (to avoid massive lists in logs).
	FirstEnrollmentID = "id_first"

	CommandUUID = "command_uuid"
	RequestType = "request_type"

	InstanceID   = "instance_id"
	WorkflowName = "workflow_name"
	StepName     = "step_name"

	// a context-dependent numerical count/length of something
	GenericCount = "count"
)
