-- name: UpdateStepAfterNotUntil :exec
UPDATE
  steps
SET
  not_until_proc = ?
WHERE
  not_until_proc IS NULL AND
  not_until < ?;

-- name: GetStepsByNotUntilProc :many
SELECT
  id,
  workflow_name,
  instance_id,
  step_name
FROM
  steps
WHERE
  not_until_proc = ?;

-- name: GetStepCommandsByNotUntilProc :many
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
  s.not_until_proc = ?;

-- name: GetIDCommandIDsByNotUntilProc :many
SELECT
  step_id,
  enrollment_id
FROM
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
WHERE
  s.not_until_proc = ?;

-- name: RemoveStepCommandsByNotUntilProc :exec
DELETE sc FROM
  step_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.not_until_proc = ?;

-- name: UpdateLastPushByNotUntilProc :exec
UPDATE
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
SET
  c.last_push = CURRENT_TIMESTAMP
WHERE
  s.not_until_proc = ?;

-- name: UpdateStepAfterTimeout :exec
UPDATE
  steps
SET
  timeout_proc = ?
WHERE
  timeout_proc IS NULL AND
  timeout <= ?;

-- name: GetStepsByTimeoutProc :many
SELECT
  id,
  workflow_name,
  instance_id,
  step_name,
  context
FROM
  steps
WHERE
  timeout_proc = ?;

-- name: GetIDCommandIDsByTimeoutProc :many
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
  step_id, enrollment_id;

-- name: RemoveStepCommandsByTimeoutProc :exec
DELETE sc FROM
  step_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.timeout_proc = ?;

-- name: RemoveIDCommandsByTimeoutProc :exec
DELETE sc FROM
  id_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.timeout_proc = ?;

-- name: RemoveStepsByTimeoutProc :exec
DELETE FROM
  steps
WHERE
  timeout_proc = ?;

-- name: GetRePushIDs :many
SELECT DISTINCT
  enrollment_id
FROM
  id_commands
WHERE
  last_push IS NOT NULL AND
  last_push < sqlc.arg(before);

-- name: UpdateRePushIDs :exec
UPDATE
  id_commands
SET
  last_push = ?
WHERE
  last_push IS NOT NULL AND
  last_push < sqlc.arg(before);
