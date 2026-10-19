-- name: CreateAutoPlay :exec
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, conditions, priority, once_within_minutes, run_on, computer_id, run_as, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAutoPlay :one
SELECT * FROM auto_plays WHERE id = ?;

-- name: ListAutoPlaysByPlays :many
SELECT * FROM auto_plays WHERE play_id IN (SELECT value FROM json_each(sqlc.arg(play_ids))) ORDER BY play_id, created_at, id;

-- name: UpdateAutoPlay :execrows
UPDATE auto_plays SET enabled = ?, moment = ?, moment_stage = ?, conditions = ?, priority = ?, once_within_minutes = ?, run_on = ?, computer_id = ?, run_as = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteAutoPlay :execrows
DELETE FROM auto_plays WHERE id = ?;

-- name: GetAutoPlayLimits :one
SELECT auto_play_daily_cap, auto_play_runs_at_once FROM workspaces WHERE id = ?;

-- name: SetAutoPlayLimits :execrows
UPDATE workspaces SET auto_play_daily_cap = ?, auto_play_runs_at_once = ? WHERE id = ?;

-- name: ListEnabledAutoPlaysByMoment :many
SELECT * FROM auto_plays WHERE workspace_id = ? AND moment = ? AND enabled = 1
  AND NOT EXISTS (SELECT 1 FROM workspaces w WHERE w.id = auto_plays.workspace_id AND w.archived_at IS NOT NULL)
ORDER BY created_at, id;
