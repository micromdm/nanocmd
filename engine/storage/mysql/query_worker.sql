-- name: UpdateStepAfterNotUntil :exec
UPDATE
  steps
SET
  process_id = ?
WHERE
  process_id IS NULL AND
  not_until < ?;

-- name: GetStepsByProcessID :many
SELECT
  id,
  workflow_name,
  instance_id,
  step_name
FROM
  steps
WHERE
  process_id = ?;

-- name: GetStepCommandsByProcessID :many
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
  s.process_id = ?;

-- name: GetIDCommandIDsByProcessID :many
SELECT
  step_id,
  enrollment_id
FROM
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
WHERE
  s.process_id = ?;

-- name: RemoveStepCommandsByProcessID :exec
DELETE sc FROM
  step_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.process_id = ?;

-- name: UpdateLastPushByProcessID :exec
UPDATE
  id_commands c
  JOIN steps s
    ON c.step_id = s.id
SET
  c.last_push = CURRENT_TIMESTAMP
WHERE
  s.process_id = ?;

-- name: UpdateStepAfterTimeout :exec
UPDATE
  steps
SET
  process_id = ?
WHERE
  process_id IS NULL AND
  timeout <= ?;

-- name: GetStepsWithContextByProcessID :many
SELECT
  id,
  workflow_name,
  instance_id,
  step_name,
  context
FROM
  steps
WHERE
  process_id = ?;

-- name: GetIDCommandDetailsByProcessID :many
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
  s.process_id = ?
ORDER BY
  step_id, enrollment_id;

-- name: RemoveIDCommandsByProcessID :exec
DELETE sc FROM
  id_commands sc
  JOIN steps s
    ON sc.step_id = s.id
WHERE
  s.process_id = ?;

-- name: RemoveStepsByProcessID :exec
DELETE FROM
  steps
WHERE
  process_id = ?;

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
