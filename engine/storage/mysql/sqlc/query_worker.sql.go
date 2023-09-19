// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.21.0
// source: query_worker.sql

package sqlc

import (
	"context"
	"database/sql"
)

const getIDCommandIDsByNotUntilProc = `-- name: GetIDCommandIDsByNotUntilProc :many
SELECT
  step_id,
  enrollment_id
FROM
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
WHERE
  s.not_until_proc = ?
`

type GetIDCommandIDsByNotUntilProcRow struct {
	StepID       int64
	EnrollmentID string
}

func (q *Queries) GetIDCommandIDsByNotUntilProc(ctx context.Context, notUntilProc sql.NullString) ([]GetIDCommandIDsByNotUntilProcRow, error) {
	rows, err := q.db.QueryContext(ctx, getIDCommandIDsByNotUntilProc, notUntilProc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetIDCommandIDsByNotUntilProcRow
	for rows.Next() {
		var i GetIDCommandIDsByNotUntilProcRow
		if err := rows.Scan(&i.StepID, &i.EnrollmentID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getIDCommandIDsByTimeoutProc = `-- name: GetIDCommandIDsByTimeoutProc :many
SELECT
  step_id,
  enrollment_id,
  command_uuid,
  request_type,
  completed,
  result
FROM
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
WHERE
  s.timeout_proc = ?
ORDER BY
  step_id, enrollment_id
`

type GetIDCommandIDsByTimeoutProcRow struct {
	StepID       int64
	EnrollmentID string
	CommandUuid  string
	RequestType  string
	Completed    bool
	Result       []byte
}

func (q *Queries) GetIDCommandIDsByTimeoutProc(ctx context.Context, timeoutProc sql.NullString) ([]GetIDCommandIDsByTimeoutProcRow, error) {
	rows, err := q.db.QueryContext(ctx, getIDCommandIDsByTimeoutProc, timeoutProc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetIDCommandIDsByTimeoutProcRow
	for rows.Next() {
		var i GetIDCommandIDsByTimeoutProcRow
		if err := rows.Scan(
			&i.StepID,
			&i.EnrollmentID,
			&i.CommandUuid,
			&i.RequestType,
			&i.Completed,
			&i.Result,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getRePushIDs = `-- name: GetRePushIDs :many
SELECT DISTINCT
  enrollment_id
FROM
  id_commands
WHERE
  last_push IS NOT NULL AND
  last_push < ?
`

func (q *Queries) GetRePushIDs(ctx context.Context, before sql.NullTime) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getRePushIDs, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var enrollment_id string
		if err := rows.Scan(&enrollment_id); err != nil {
			return nil, err
		}
		items = append(items, enrollment_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStepCommandsByNotUntilProc = `-- name: GetStepCommandsByNotUntilProc :many
SELECT
  sc.step_id,
  sc.command_uuid,
  sc.request_type,
  sc.command
FROM
  step_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.not_until_proc = ?
`

type GetStepCommandsByNotUntilProcRow struct {
	StepID      int64
	CommandUuid string
	RequestType string
	Command     []byte
}

func (q *Queries) GetStepCommandsByNotUntilProc(ctx context.Context, notUntilProc sql.NullString) ([]GetStepCommandsByNotUntilProcRow, error) {
	rows, err := q.db.QueryContext(ctx, getStepCommandsByNotUntilProc, notUntilProc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStepCommandsByNotUntilProcRow
	for rows.Next() {
		var i GetStepCommandsByNotUntilProcRow
		if err := rows.Scan(
			&i.StepID,
			&i.CommandUuid,
			&i.RequestType,
			&i.Command,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStepsByNotUntilProc = `-- name: GetStepsByNotUntilProc :many
SELECT
  id,
  workflow_name,
  instance_id,
  step_name
FROM
  steps
WHERE
  not_until_proc = ?
`

type GetStepsByNotUntilProcRow struct {
	ID           int64
	WorkflowName string
	InstanceID   string
	StepName     sql.NullString
}

func (q *Queries) GetStepsByNotUntilProc(ctx context.Context, notUntilProc sql.NullString) ([]GetStepsByNotUntilProcRow, error) {
	rows, err := q.db.QueryContext(ctx, getStepsByNotUntilProc, notUntilProc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStepsByNotUntilProcRow
	for rows.Next() {
		var i GetStepsByNotUntilProcRow
		if err := rows.Scan(
			&i.ID,
			&i.WorkflowName,
			&i.InstanceID,
			&i.StepName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getStepsByTimeoutProc = `-- name: GetStepsByTimeoutProc :many
SELECT
  id,
  workflow_name,
  instance_id,
  step_name,
  context
FROM
  steps
WHERE
  timeout_proc = ?
`

type GetStepsByTimeoutProcRow struct {
	ID           int64
	WorkflowName string
	InstanceID   string
	StepName     sql.NullString
	Context      []byte
}

func (q *Queries) GetStepsByTimeoutProc(ctx context.Context, timeoutProc sql.NullString) ([]GetStepsByTimeoutProcRow, error) {
	rows, err := q.db.QueryContext(ctx, getStepsByTimeoutProc, timeoutProc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStepsByTimeoutProcRow
	for rows.Next() {
		var i GetStepsByTimeoutProcRow
		if err := rows.Scan(
			&i.ID,
			&i.WorkflowName,
			&i.InstanceID,
			&i.StepName,
			&i.Context,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeIDCommandsByTimeoutProc = `-- name: RemoveIDCommandsByTimeoutProc :exec
DELETE sc FROM
  id_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.timeout_proc = ?
`

func (q *Queries) RemoveIDCommandsByTimeoutProc(ctx context.Context, timeoutProc sql.NullString) error {
	_, err := q.db.ExecContext(ctx, removeIDCommandsByTimeoutProc, timeoutProc)
	return err
}

const removeStepCommandsByNotUntilProc = `-- name: RemoveStepCommandsByNotUntilProc :exec
DELETE sc FROM
  step_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.not_until_proc = ?
`

func (q *Queries) RemoveStepCommandsByNotUntilProc(ctx context.Context, notUntilProc sql.NullString) error {
	_, err := q.db.ExecContext(ctx, removeStepCommandsByNotUntilProc, notUntilProc)
	return err
}

const removeStepCommandsByTimeoutProc = `-- name: RemoveStepCommandsByTimeoutProc :exec
DELETE sc FROM
  step_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.timeout_proc = ?
`

func (q *Queries) RemoveStepCommandsByTimeoutProc(ctx context.Context, timeoutProc sql.NullString) error {
	_, err := q.db.ExecContext(ctx, removeStepCommandsByTimeoutProc, timeoutProc)
	return err
}

const removeStepsByTimeoutProc = `-- name: RemoveStepsByTimeoutProc :exec
DELETE FROM
  steps
WHERE
  timeout_proc = ?
`

func (q *Queries) RemoveStepsByTimeoutProc(ctx context.Context, timeoutProc sql.NullString) error {
	_, err := q.db.ExecContext(ctx, removeStepsByTimeoutProc, timeoutProc)
	return err
}

const updateLastPushByNotUntilProc = `-- name: UpdateLastPushByNotUntilProc :exec
UPDATE
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
SET
  c.last_push = CURRENT_TIMESTAMP
WHERE
  s.not_until_proc = ?
`

func (q *Queries) UpdateLastPushByNotUntilProc(ctx context.Context, notUntilProc sql.NullString) error {
	_, err := q.db.ExecContext(ctx, updateLastPushByNotUntilProc, notUntilProc)
	return err
}

const updateRePushIDs = `-- name: UpdateRePushIDs :exec
UPDATE
  id_commands
SET
  last_push = ?
WHERE
  last_push IS NOT NULL AND
  last_push < ?
`

type UpdateRePushIDsParams struct {
	LastPush sql.NullTime
	Before   sql.NullTime
}

func (q *Queries) UpdateRePushIDs(ctx context.Context, arg UpdateRePushIDsParams) error {
	_, err := q.db.ExecContext(ctx, updateRePushIDs, arg.LastPush, arg.Before)
	return err
}

const updateStepAfterNotUntil = `-- name: UpdateStepAfterNotUntil :exec
UPDATE
  steps
SET
  not_until_proc = ?
WHERE
  not_until_proc IS NULL AND
  not_until < ?
`

type UpdateStepAfterNotUntilParams struct {
	NotUntilProc sql.NullString
	NotUntil     sql.NullTime
}

func (q *Queries) UpdateStepAfterNotUntil(ctx context.Context, arg UpdateStepAfterNotUntilParams) error {
	_, err := q.db.ExecContext(ctx, updateStepAfterNotUntil, arg.NotUntilProc, arg.NotUntil)
	return err
}

const updateStepAfterTimeout = `-- name: UpdateStepAfterTimeout :exec
UPDATE
  steps
SET
  timeout_proc = ?
WHERE
  timeout_proc IS NULL AND
  timeout <= ?
`

type UpdateStepAfterTimeoutParams struct {
	TimeoutProc sql.NullString
	Timeout     sql.NullTime
}

func (q *Queries) UpdateStepAfterTimeout(ctx context.Context, arg UpdateStepAfterTimeoutParams) error {
	_, err := q.db.ExecContext(ctx, updateStepAfterTimeout, arg.TimeoutProc, arg.Timeout)
	return err
}
