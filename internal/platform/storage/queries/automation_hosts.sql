-- name: GetAutomationHost :one
SELECT * FROM automation_hosts WHERE id = ?;

-- name: GetAutomationHostByName :one
SELECT * FROM automation_hosts WHERE name = ?;

-- name: ListAutomationHosts :many
SELECT * FROM automation_hosts ORDER BY created_at, id;

-- name: CreateAutomationHost :exec
INSERT INTO automation_hosts (id, name, machine, os, arch, version, last_seen, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: TouchAutomationHost :execrows
UPDATE automation_hosts SET last_seen = ? WHERE id = ?;

-- name: DeleteAutomationHost :execrows
DELETE FROM automation_hosts WHERE id = ?;

-- name: PruneAutomationHostEnrollmentCodes :exec
DELETE FROM automation_host_enrollment_codes WHERE expires_at <= ?;

-- name: CreateAutomationHostEnrollmentCode :exec
INSERT INTO automation_host_enrollment_codes (code_hash, name, machine, created_at, expires_at) VALUES (?, ?, ?, ?, ?);

-- name: GetAutomationHostEnrollmentCode :one
SELECT * FROM automation_host_enrollment_codes WHERE code_hash = ? AND expires_at > ?;

-- name: ConsumeAutomationHostEnrollmentCode :execrows
DELETE FROM automation_host_enrollment_codes WHERE code_hash = ? AND expires_at > ?;

-- name: CreateAutomationHostCredential :exec
INSERT INTO automation_host_credentials (credential_hash, host_id, host_name, created_at) VALUES (?, ?, ?, ?);

-- name: GetAutomationHostCredential :one
SELECT * FROM automation_host_credentials WHERE credential_hash = ?;

-- name: GetLiveAutomationHostCredential :one
SELECT * FROM automation_host_credentials WHERE host_id = ? AND revoked_at IS NULL LIMIT 1;

-- name: RevokeAutomationHostCredentials :exec
UPDATE automation_host_credentials SET revoked_at = ? WHERE host_id = ? AND revoked_at IS NULL;
