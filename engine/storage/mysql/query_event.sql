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

-- name: GetEventsByNames :many
SELECT
  event_name,
  context,
  workflow_name,
  event_type
FROM
  wf_events
WHERE
  event_name IN (sqlc.slice('names'));

-- name: GetEventsByType :many
SELECT
  context,
  workflow_name,
  event_type
FROM
  wf_events
WHERE
  event_type = ?;

-- name: RemoveEvent :exec
DELETE FROM wf_events WHERE event_name = ?;
