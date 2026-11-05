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

-- The only INSERT INTO id_commands. StoreStep always creates the steps row
-- first, so an existing step can never gain a reference -- which is what lets
-- SelectUnreferencedStepIDs run without a lock. Do not add a path that points
-- new id_commands at an existing step.
-- name: CreateIDCommand :exec
INSERT INTO id_commands
  (enrollment_id, command_uuid, step_id, request_type, last_push)
VALUES
  (?, ?, ?, ?, ?);

-- Scoped by workflow name rather than by a list of step IDs, so the statement
-- is a fixed size: an enrollment's step IDs are unbounded, and 65535
-- placeholders is a hard limit.
--
-- MySQL 8 semi-joins this. For an enrollment holding few rows it drives from
-- id_commands and does one primary key lookup per row; for one holding very
-- many it flips to scanning the workflow_name index instead. Measured against
-- 100k steps sharing a name: 0.04ms for a 5-row enrollment, 337ms for a
-- 50,000-row one.
-- name: DeleteIDCommandsByWorkflowName :exec
DELETE FROM
  id_commands
WHERE
  enrollment_id = ? AND
  step_id IN (
    SELECT
      id
    FROM
      steps
    WHERE
      workflow_name = ?
  );

-- name: DeleteIDCommands :exec
DELETE FROM
  id_commands
WHERE
  enrollment_id = ?;

-- name: GetStepIDsByEnrollmentID :many
SELECT DISTINCT
  step_id
FROM
  id_commands
WHERE
  enrollment_id = ?;

-- name: GetStepIDsByEnrollmentIDAndWorkflowName :many
SELECT DISTINCT
  c.step_id
FROM
  id_commands c
  INNER JOIN steps s
    ON c.step_id = s.id
WHERE
  c.enrollment_id = ? AND
  s.workflow_name = ?;

-- Reports which of ids nothing refers to any more. Needs no lock, because
-- "unreferenced" is a terminal state: see CreateIDCommand.
--
-- Must run outside a transaction, after the caller's own deletes have
-- committed. Under REPEATABLE READ a caller asking this mid-transaction is
-- served from its own snapshot, so its peers' committed deletes are invisible
-- to it and every one of them concludes the step is still in use.
-- name: SelectUnreferencedStepIDs :many
SELECT
  s.id
FROM
  steps s
WHERE
  s.id IN (sqlc.slice('ids')) AND
  NOT EXISTS (
    SELECT
      1
    FROM
      id_commands c
    WHERE
      c.step_id = s.id
  );

-- Claims steps for collection. Rows another collector already holds are not
-- returned, so nobody ever waits here: whoever arrives first collects, the
-- rest move on and leave the step to them.
-- name: SelectStepIDsForDelete :many
SELECT
  id
FROM
  steps
WHERE
  id IN (sqlc.slice('ids'))
ORDER BY
  id
FOR UPDATE SKIP LOCKED;

-- name: DeleteStepCommandsByStepIDs :exec
DELETE FROM
  step_commands
WHERE
  step_id IN (sqlc.slice('step_ids'));

-- name: DeleteStepsByStepIDs :exec
DELETE FROM
  steps
WHERE
  id IN (sqlc.slice('ids'));

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

-- name: GetStepIDByCommandUUID :one
SELECT
  step_id
FROM
  id_commands
WHERE
  enrollment_id = ? AND
  command_uuid = ?;

-- Reads and locks this enrollment's rows for the step, and only those. Step
-- completion is decided from this result, so it has to lock: the worker's
-- timeout sweep deletes these same rows on a timer, serialized with nothing
-- (see RetrieveTimedOutSteps). No other enrollment owns them, so nothing
-- else ever waits here.
--
-- Do not add joins. "step_id = ?" already says what a steps join would, and a
-- steps join under FOR UPDATE locks the one row the whole fan-out shares.
-- name: GetIDCommandsByStepIDAndLock :many
SELECT
  command_uuid,
  request_type,
  result,
  completed
FROM
  id_commands
WHERE
  enrollment_id = ? AND
  step_id = ?
FOR UPDATE;

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
  last_created_unix
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
