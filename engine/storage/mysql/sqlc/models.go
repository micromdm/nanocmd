// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.21.0

package sqlc

import (
	"database/sql"
)

type IDCommand struct {
	EnrollmentID string
	CommandUuid  string
	StepID       int64
	RequestType  string
	Completed    bool
	Result       []byte
	LastPush     sql.NullTime
	CreatedAt    sql.NullTime
	UpdatedAt    sql.NullTime
}

type Step struct {
	ID           int64
	WorkflowName string
	InstanceID   string
	StepName     sql.NullString
	Context      []byte
	NotUntil     sql.NullTime
	NotUntilProc sql.NullString
	Timeout      sql.NullTime
	TimeoutProc  sql.NullString
	CreatedAt    sql.NullTime
	UpdatedAt    sql.NullTime
}

type StepCommand struct {
	CommandUuid string
	StepID      int64
	Command     []byte
	RequestType string
	CreatedAt   sql.NullTime
	UpdatedAt   sql.NullTime
}

type WfEvent struct {
	EventName    string
	Context      sql.NullString
	WorkflowName string
	EventType    string
	CreatedAt    sql.NullTime
	UpdatedAt    sql.NullTime
}
