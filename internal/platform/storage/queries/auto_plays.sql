-- name: CreateAutoPlay :exec
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, conditions, priority, once_within_minutes, run_on, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAutoPlay :one
SELECT * FROM auto_plays WHERE id = ?;

-- name: ListAutoPlaysByPlays :many
SELECT * FROM auto_plays WHERE play_id IN (SELECT value FROM json_each(sqlc.arg(play_ids))) ORDER BY play_id, created_at, id;

-- name: UpdateAutoPlay :execrows
UPDATE auto_plays SET enabled = ?, moment = ?, moment_stage = ?, conditions = ?, priority = ?, once_within_minutes = ?, run_on = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteAutoPlay :execrows
DELETE FROM auto_plays WHERE id = ?;

-- name: GetAutoPlayDailyCap :one
SELECT auto_play_daily_cap FROM workspaces WHERE id = ?;

-- name: SetAutoPlayDailyCap :execrows
UPDATE workspaces SET auto_play_daily_cap = ? WHERE id = ?;
