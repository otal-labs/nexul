-- name: PublishSchema :exec
INSERT OR IGNORE INTO event_schemas (topic, version, schema, created_at) VALUES (?, ?, ?, ?);

-- name: ListSchemaCatalog :many
SELECT * FROM event_schemas ORDER BY topic, version DESC;

-- name: AppendAudit :exec
INSERT INTO audit_log (id, actor_type, actor_id, token_id, action, created_at) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListAudit :many
SELECT * FROM audit_log ORDER BY created_at DESC LIMIT ?;
