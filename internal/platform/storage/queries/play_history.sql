-- name: InsertPlayInstructionVersion :exec
INSERT INTO play_instruction_versions (play_id, version, body, editor_ids, via, builtin_default, words_added, words_removed, restored_from, created_at)
VALUES (
    sqlc.arg(play_id),
    (SELECT COALESCE(MAX(v.version), 0) + 1 FROM play_instruction_versions v WHERE v.play_id = sqlc.arg(play_id)),
    sqlc.arg(body), sqlc.arg(editor_ids), sqlc.arg(via), sqlc.arg(builtin_default), sqlc.arg(words_added), sqlc.arg(words_removed), sqlc.arg(restored_from), sqlc.arg(created_at)
);

-- name: PrunePlayInstructionVersions :exec
DELETE FROM play_instruction_versions
WHERE play_instruction_versions.play_id = sqlc.arg(play_id)
  AND version <= (SELECT MAX(v.version) FROM play_instruction_versions v WHERE v.play_id = sqlc.arg(play_id)) - sqlc.arg(keep);

-- name: ListPlayInstructionVersions :many
SELECT play_id, version, editor_ids, via, builtin_default, words_added, words_removed, restored_from, created_at FROM play_instruction_versions
WHERE play_id = ? ORDER BY version DESC;

-- name: ListPlayInstructionVersionNumbers :many
SELECT version FROM play_instruction_versions WHERE play_id = ? ORDER BY version;

-- name: SetPlayInstructionVersionSize :exec
UPDATE play_instruction_versions SET words_added = ?, words_removed = ? WHERE play_id = ? AND version = ?;

-- name: GetPlayInstructionVersion :one
SELECT * FROM play_instruction_versions WHERE play_id = ? AND version = ?;

-- name: InsertPlayChange :exec
INSERT INTO play_changes (id, play_id, field, from_value, to_value, actor, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: PrunePlayChanges :exec
DELETE FROM play_changes
WHERE play_changes.play_id = sqlc.arg(play_id)
  AND id NOT IN (SELECT c.id FROM play_changes c WHERE c.play_id = sqlc.arg(play_id) ORDER BY c.created_at DESC, c.id DESC LIMIT sqlc.arg(keep));

-- name: ListPlayChanges :many
SELECT * FROM play_changes WHERE play_id = ? ORDER BY created_at DESC, id DESC;
