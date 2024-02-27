-- name: GetEventsByNames :many
SELECT
  event_name,
  context,
  event_context,
  workflow_name,
  event_type
FROM
  wf_events
WHERE
  event_name IN (sqlc.slice('names'));

-- name: GetEventsByType :many
SELECT
  context,
  event_context,
  workflow_name,
  event_type
FROM
  wf_events
WHERE
  event_type = ?;

-- name: RemoveEvent :exec
DELETE FROM wf_events WHERE event_name = ?;

