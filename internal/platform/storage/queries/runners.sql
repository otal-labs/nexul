-- name: CreateRunner :exec
INSERT INTO runners (id, name, version, last_seen, connected, created_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetRunner :one
SELECT * FROM runners WHERE id = ?;

-- name: ListRunners :many
SELECT * FROM runners ORDER BY created_at;

-- name: SetRunnerVersion :execrows
UPDATE runners SET version = ? WHERE id = ?;

-- name: UpdateRunnerHeartbeat :execrows
UPDATE runners SET last_seen = ?, connected = ? WHERE id = ?;

-- name: DeleteRunner :execrows
DELETE FROM runners WHERE id = ?;

-- name: SetRunnerMachine :execrows
UPDATE runners SET machine_id = ? WHERE id = ?;

-- name: GetRunnerSecret :one
SELECT runner_secret FROM instance_settings WHERE id = 1;

-- name: SeedRunnerSecret :exec
UPDATE instance_settings SET runner_secret = ? WHERE id = 1 AND runner_secret = '';

-- name: SetRunnerSecret :exec
UPDATE instance_settings SET runner_secret = ? WHERE id = 1;

-- name: CreateInstanceUpgrade :exec
INSERT INTO instance_upgrades (id, from_version, to_version, status, error, requested_by, runner_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetInstanceUpgrade :one
SELECT * FROM instance_upgrades WHERE id = ?;

-- name: GetLatestInstanceUpgrade :one
SELECT * FROM instance_upgrades ORDER BY created_at DESC LIMIT 1;

-- name: ListUnresolvedInstanceUpgrades :many
SELECT * FROM instance_upgrades WHERE status IN ('pending', 'started') ORDER BY created_at;

-- name: SetInstanceUpgradeStatus :execrows
UPDATE instance_upgrades SET status = ?, error = ?, updated_at = ? WHERE id = ?;
