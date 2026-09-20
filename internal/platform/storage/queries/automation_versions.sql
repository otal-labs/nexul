-- name: InsertAutomationVersion :exec
INSERT INTO automation_versions (id, automation_id, sequence, code, pusher_id, message, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePendingAutomationVersion :exec
DELETE FROM automation_versions WHERE automation_id = ? AND status = ?;

-- name: CountAutomationVersionForAutomation :one
SELECT COUNT(*) FROM automation_versions WHERE id = ? AND automation_id = ?;

-- name: DeactivateAutomationVersions :exec
UPDATE automation_versions SET status = sqlc.arg(new_status) WHERE automation_id = sqlc.arg(automation_id) AND status = sqlc.arg(old_status);

-- name: SetAutomationVersionStatus :exec
UPDATE automation_versions SET status = ? WHERE id = ?;

-- name: GetAutomationVersionByID :one
SELECT * FROM automation_versions WHERE id = ?;

-- name: GetAutomationVersion :one
SELECT * FROM automation_versions WHERE id = ? AND automation_id = ?;

-- name: ListAutomationVersionsByAutomation :many
SELECT * FROM automation_versions WHERE automation_id = ? ORDER BY sequence DESC;

-- name: GetAutomationVersionByStatus :one
SELECT * FROM automation_versions WHERE automation_id = ? AND status = ?;

-- name: MaxAutomationVersionSequence :one
SELECT MAX(sequence) FROM automation_versions WHERE automation_id = ?;
