-- name: GetAllProfileInfos :many
SELECT
  name,
  profile_id,
  profile_uuid
FROM
  subsystem_profiles;

-- name: GetProfileInfos :many
SELECT
  name,
  profile_id,
  profile_uuid
FROM
  subsystem_profiles
WHERE
  name IN (sqlc.slice('names'));

-- name: GetRawProfiles :many
SELECT
  name,
  raw_profile
FROM
  subsystem_profiles
WHERE
  name IN (sqlc.slice('names'));

-- name: DeleteProfile :exec
DELETE FROM subsystem_profiles WHERE name = ?;
