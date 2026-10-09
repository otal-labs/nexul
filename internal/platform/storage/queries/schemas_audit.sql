-- name: PublishSchema :exec
INSERT OR IGNORE INTO event_schemas (topic, version, schema, created_at) VALUES (?, ?, ?, ?);

-- name: ListSchemaCatalog :many
SELECT * FROM event_schemas ORDER BY topic, version DESC;

-- name: AppendAudit :exec
INSERT INTO audit_log (id, actor_type, actor_id, token_id, action, created_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListAudit :many
SELECT * FROM audit_log ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: DeleteAuditBefore :execrows
DELETE FROM audit_log WHERE id IN (SELECT a.id FROM audit_log AS a WHERE a.created_at < sqlc.arg(before) LIMIT sqlc.arg(max_rows));
