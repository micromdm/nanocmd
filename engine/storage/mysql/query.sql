-- name: GetRequestType :one
SELECT
  request_type
FROM
  id_commands
WHERE
  enrollment_id = ? AND
  command_uuid = ?;

-- name: CreateStep :execlastid
INSERT INTO steps
  (workflow_name, instance_id, step_name, context, not_until, timeout)
VALUES
  (?, ?, ?, ?, ?, ?);

-- name: CreateIDCommand :exec
INSERT INTO id_commands
  (enrollment_id, command_uuid, step_id, request_type, last_push)
VALUES
  (?, ?, ?, ?, ?);

-- name: DeleteIDCommandByWorkflow :exec
DELETE
  c
FROM
  id_commands c
  INNER JOIN steps s
    ON c.step_id = s.id
WHERE
  c.enrollment_id = ? AND
  s.workflow_name = ?;

-- name: DeleteIDCommands :exec
DELETE FROM
  id_commands
WHERE
  enrollment_id = ?;

-- name: DeleteUnusedStepCommands :exec
DELETE
  sc
FROM
  step_commands sc
  LEFT JOIN id_commands c
    ON sc.command_uuid = c.command_uuid
WHERE
  c.command_uuid IS NULL;

-- name: DeleteWorkflowStepHavingNoCommands :exec
DELETE
  s
FROM
  steps s
  LEFT JOIN id_commands c
    ON s.id = c.step_id
WHERE
  c.step_id IS NULL;

-- name: DeleteWorkflowStepHavingNoCommandsByWorkflowName :exec
DELETE
  s
FROM
  steps s
  LEFT JOIN id_commands c
    ON s.id = c.step_id
WHERE
  c.step_id IS NULL AND
  s.workflow_name = ?;

-- name: UpdateIDCommandTimestamp :exec
UPDATE
  id_commands
SET
  updated_at = CURRENT_TIMESTAMP
WHERE
  enrollment_id = ? AND
  command_uuid = ?
LIMIT 1;

-- name: UpdateIDCommand :exec
UPDATE
  id_commands
SET
  completed = ?,
  result = ?
WHERE
  enrollment_id = ? AND
  command_uuid = ?
LIMIT 1;

-- name: CountOutstandingIDWorkflowStepCommands :one
SELECT
  COUNT(*),
  c1.step_id
FROM
  id_commands c1
  JOIN id_commands c2
    ON c1.step_id = c2.step_id
WHERE
  c1.enrollment_id = ? AND
  c1.completed = 0 AND
  c2.enrollment_id = c1.enrollment_id AND
  c2.command_uuid = ?
GROUP BY
  c1.step_id
LIMIT 1;

-- name: GetStepByID :one
SELECT
  workflow_name,
  instance_id,
  step_name,
  context
FROM
  steps
WHERE
  id = ?;

-- name: GetIDCommandsByStepID :many
SELECT
  command_uuid,
  request_type,
  result
FROM
  id_commands
WHERE
  enrollment_id = ? AND
  step_id = ? AND
  completed != 0;

-- name: RemoveIDCommandsByStepID :exec
DELETE FROM
  id_commands
WHERE
  enrollment_id = ? AND
  step_id = ?;

-- name: CreateStepCommand :exec
INSERT INTO step_commands
  (step_id, command_uuid, request_type, command)
VALUES
  (?, ?, ?, ?);

-- name: GetOutstandingIDs :many
SELECT DISTINCT
  c.enrollment_id
FROM
  id_commands c
  JOIN steps s
    ON s.id = c.step_id
WHERE
  c.enrollment_id IN (sqlc.slice('ids')) AND
  c.completed = 0 AND
  s.workflow_name = ?;

-- name: GetWorkflowLastStarted :one
SELECT
  last_created_at
FROM
  wf_status
WHERE
  enrollment_id = ? AND
  workflow_name = ?;

-- name: ClearWorkflowStatus :exec
DELETE FROM
  wf_status
WHERE
  enrollment_id = ?;
