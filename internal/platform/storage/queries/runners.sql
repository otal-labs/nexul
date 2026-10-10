-- name: CreateRunner :exec
INSERT INTO runners (id, name, version, last_seen, connected, created_at, machine_id, owner_user_id, computer_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetRunner :one
SELECT * FROM runners WHERE id = ?;

-- name: ListRunners :many
SELECT * FROM runners WHERE owner_user_id = '' ORDER BY created_at;

-- name: SetRunnerVersion :execrows
UPDATE runners SET version = ? WHERE id = ?;

-- name: UpdateRunnerHeartbeat :execrows
UPDATE runners SET last_seen = ?, connected = ? WHERE id = ?;

-- name: DeleteRunner :execrows
DELETE FROM runners WHERE id = ?;

-- name: GetRunnerByName :one
SELECT * FROM runners WHERE name = ?;

-- name: GetRunnerByComputer :one
SELECT * FROM runners WHERE computer_id = ? AND computer_id != '' LIMIT 1;

-- name: PruneRunnerEnrollmentCodes :exec
DELETE FROM runner_enrollment_codes WHERE expires_at <= ?;

-- name: CreateRunnerEnrollmentCode :exec
INSERT INTO runner_enrollment_codes (code_hash, name, machine, created_at, expires_at, owner_user_id, computer_id) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetRunnerEnrollmentCode :one
SELECT * FROM runner_enrollment_codes WHERE code_hash = ? AND expires_at > ?;

-- name: ConsumeRunnerEnrollmentCode :execrows
DELETE FROM runner_enrollment_codes WHERE code_hash = ? AND expires_at > ?;

-- name: CreateRunnerCredential :exec
INSERT INTO runner_credentials (credential_hash, runner_id, runner_name, created_at) VALUES (?, ?, ?, ?);

-- name: GetRunnerCredential :one
SELECT * FROM runner_credentials WHERE credential_hash = ?;

-- name: RevokeRunnerCredentials :exec
UPDATE runner_credentials SET revoked_at = ? WHERE runner_id = ? AND revoked_at IS NULL;

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
