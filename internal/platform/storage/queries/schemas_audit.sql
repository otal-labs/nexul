-- name: PublishSchema :exec
INSERT OR IGNORE INTO event_schemas (topic, version, schema, created_at) VALUES (?, ?, ?, ?);

-- name: ListSchemaCatalog :many
SELECT * FROM event_schemas ORDER BY topic, version DESC;

-- name: AppendAudit :exec
INSERT INTO audit_log (id, actor_type, actor_id, token_id, action, created_at, computer_id, computer_owner_id, job_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAudit :many
-- An action on a person's computer is computer activity, never in the audit log itself.
SELECT * FROM audit_log WHERE computer_id = '' ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: ListComputerActivity :many
-- Actions on people's computers, newest first, each with the computer and its owner as they are named now and, for a
-- command, the command line, how it ended and its output while it is kept.
SELECT sqlc.embed(a),
    COALESCE(c.name, '') AS computer_name,
    COALESCE(NULLIF(o.name, ''), o.login, '') AS owner_name,
    COALESCE(j.command, '') AS command,
    COALESCE(j.status, '') AS job_status,
    j.exit_code AS exit_code,
    COALESCE(j.output, '') AS output
FROM audit_log a
LEFT JOIN pairing_computers c ON c.id = a.computer_id
LEFT JOIN users o ON o.id = a.computer_owner_id
LEFT JOIN runner_shell_jobs j ON j.id = a.job_id
WHERE a.computer_id != ''
ORDER BY a.created_at DESC, a.id DESC LIMIT ?;

-- name: DeleteAuditBefore :execrows
DELETE FROM audit_log WHERE id IN (SELECT a.id FROM audit_log AS a WHERE a.created_at < sqlc.arg(before) LIMIT sqlc.arg(max_rows));
