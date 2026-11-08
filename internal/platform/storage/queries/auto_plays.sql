-- name: CreateAutoPlay :exec
INSERT INTO auto_plays (id, play_id, workspace_id, enabled, moment, moment_stage, schedule, conditions, priority, once_within_minutes, run_on, computer_id, computer_ids, run_as, memory_ids, one_at_a_time, resource, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAutoPlay :one
SELECT * FROM auto_plays WHERE id = ?;

-- name: ListAutoPlaysByPlays :many
SELECT * FROM auto_plays WHERE play_id IN (SELECT value FROM json_each(sqlc.arg(play_ids))) ORDER BY play_id, created_at, id;

-- name: UpdateAutoPlay :execrows
UPDATE auto_plays SET enabled = ?, moment = ?, moment_stage = ?, schedule = ?, conditions = ?, priority = ?, once_within_minutes = ?, run_on = ?, computer_id = ?, computer_ids = ?, run_as = ?, memory_ids = ?, one_at_a_time = ?, resource = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteAutoPlay :execrows
DELETE FROM auto_plays WHERE id = ?;

-- name: GetAutoPlayLimits :one
SELECT auto_play_daily_cap FROM workspaces WHERE id = ?;

-- name: SetAutoPlayLimits :execrows
UPDATE workspaces SET auto_play_daily_cap = ? WHERE id = ?;

-- name: ListEnabledAutoPlaysByMoment :many
SELECT * FROM auto_plays WHERE workspace_id = ? AND moment = ? AND enabled = 1
  AND NOT EXISTS (SELECT 1 FROM workspaces w WHERE w.id = auto_plays.workspace_id AND w.archived_at IS NOT NULL)
ORDER BY created_at, id;

-- name: ListEnabledScheduleAutoPlays :many
SELECT * FROM auto_plays WHERE moment = 'schedule' AND enabled = 1
  AND NOT EXISTS (SELECT 1 FROM workspaces w WHERE w.id = auto_plays.workspace_id AND w.archived_at IS NOT NULL)
ORDER BY created_at, id;

-- name: InsertAutoPlaySweep :execrows
INSERT INTO auto_play_sweeps (id, workspace_id, auto_play_id, play_id, play_label, fired_at, start_by, matched, over_cap, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT DO NOTHING;

-- name: GetAutoPlaySweep :one
SELECT * FROM auto_play_sweeps WHERE id = ?;

-- name: ListLastAutoPlaySweeps :many
SELECT * FROM auto_play_sweeps WHERE id IN (
    SELECT (SELECT s.id FROM auto_play_sweeps s WHERE s.auto_play_id = j.value ORDER BY s.fired_at DESC, s.id DESC LIMIT 1)
    FROM json_each(sqlc.arg(auto_play_ids)) j
);

-- name: ListOpenAutoPlaySweeps :many
SELECT * FROM auto_play_sweeps WHERE finished_at IS NULL ORDER BY fired_at, id;

-- name: FinishAutoPlaySweep :execrows
UPDATE auto_play_sweeps SET finished_at = ?, counts = ? WHERE id = ? AND finished_at IS NULL;
