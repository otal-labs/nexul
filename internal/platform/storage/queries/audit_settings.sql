-- name: GetAuditSettings :one
SELECT * FROM audit_settings WHERE id = 1;

-- name: SetAuditSettings :exec
INSERT INTO audit_settings (id, retention_days, stream_url, stream_secret, stream_enabled, updated_by, updated_at)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET retention_days = excluded.retention_days, stream_url = excluded.stream_url,
    stream_secret = excluded.stream_secret, stream_enabled = excluded.stream_enabled, updated_by = excluded.updated_by,
    updated_at = excluded.updated_at;

-- name: DeleteAuditSettings :exec
DELETE FROM audit_settings WHERE id = 1;
