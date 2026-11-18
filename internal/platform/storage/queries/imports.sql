-- name: CreateImport :exec
INSERT INTO imports (id, workspace_id, project_id, source, source_from, state, created_by, credential_encrypted, total, done, failed, report_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetImport :one
SELECT * FROM imports WHERE id = ?;

-- name: ListImportsByWorkspace :many
SELECT * FROM imports WHERE workspace_id = ? ORDER BY created_at DESC, id DESC LIMIT ?;

-- name: ListOpenImports :many
SELECT * FROM imports WHERE state IN ('scanning', 'ready', 'importing') ORDER BY created_at, id;

-- name: UpdateImport :execrows
UPDATE imports SET state = ?, source_from = ?, total = ?, done = ?, failed = ?, report_json = ?, updated_at = ?
WHERE id = sqlc.arg(id) AND state = sqlc.arg(from_state);

-- name: MoveImport :execrows
-- Moves the state alone, so a cancel never writes back counts or a report older than the loop's last save.
UPDATE imports SET state = ?, updated_at = ? WHERE id = sqlc.arg(id) AND state = sqlc.arg(from_state);

-- name: SetImportError :exec
UPDATE imports SET report_json = json_set(report_json, '$.error', ?) WHERE id = ?;

-- name: ClearImportCredential :exec
-- Runs before a move to a final state, which the table's CHECK refuses while a credential is left.
UPDATE imports SET credential_encrypted = NULL WHERE id = ? AND state = ?;

-- name: GetImportItem :one
SELECT * FROM import_items WHERE import_id = ? AND kind = ? AND source_id = ?;

-- name: PutImportItem :exec
INSERT INTO import_items (import_id, kind, source_id, nexul_id, state, error) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (import_id, kind, source_id) DO UPDATE SET nexul_id = excluded.nexul_id, state = excluded.state, error = excluded.error;

-- name: GetImportSourceID :one
SELECT nexul_id FROM import_source_ids WHERE workspace_id = ? AND source = ? AND kind = ? AND source_id = ?;

-- name: PutImportSourceID :exec
INSERT INTO import_source_ids (workspace_id, source, kind, source_id, nexul_id) VALUES (?, ?, ?, ?, ?)
ON CONFLICT (workspace_id, source, kind, source_id) DO UPDATE SET nexul_id = excluded.nexul_id;

-- name: ListImportPeople :many
-- Every member of a workspace with each sign-in linked to their account, for matching an import's people.
SELECT u.id AS user_id, u.login AS user_login, i.provider, i.login AS identity_login
FROM workspace_members m JOIN users u ON u.id = m.user_id JOIN user_identities i ON i.user_id = u.id
WHERE m.workspace_id = ?
ORDER BY u.id, i.provider;

-- name: ListImportItemsInState :many
SELECT * FROM import_items WHERE import_id = ? AND kind = ? AND state = ? ORDER BY source_id;

-- name: DeleteImportSourceID :exec
DELETE FROM import_source_ids WHERE workspace_id = ? AND source = ? AND kind = ? AND source_id = ?;
