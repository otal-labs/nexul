-- name: GetInstanceTemplate :one
SELECT * FROM instance_templates WHERE kind = ? AND key = ?;

-- name: ListInstanceTemplates :many
SELECT * FROM instance_templates ORDER BY kind, key;

-- name: UpsertInstanceTemplate :exec
INSERT INTO instance_templates (kind, key, body, updated_by, updated_at) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (kind, key) DO UPDATE SET body = excluded.body, updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: DeleteInstanceTemplate :exec
DELETE FROM instance_templates WHERE kind = ? AND key = ?;
