-- name: InsertPlayQueueItem :execrows
INSERT INTO play_queue (id, workspace_id, project_id, target_type, target_id, play_id, play_label, auto_play_id, person_id, run_on, moment, via, priority, status, reason, trail_id, queued_at, decided_at, not_before)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT DO NOTHING;

-- name: GetPlayQueueItem :one
SELECT * FROM play_queue WHERE id = ?;

-- name: UpdatePlayQueueItem :execrows
UPDATE play_queue SET status = ?, reason = ?, trail_id = ?, decided_at = ?, not_before = ?
WHERE id = ? AND status = sqlc.arg(from_status);

-- name: ListPlayQueueByTarget :many
SELECT * FROM play_queue WHERE target_type = ? AND target_id = ? ORDER BY queued_at DESC, id DESC;

-- name: ListQueuedPlayQueueByPlay :many
SELECT * FROM play_queue
WHERE status = 'queued' AND play_id = sqlc.arg(play_id)
  AND (sqlc.arg(all_projects) OR project_id IN (SELECT value FROM json_each(sqlc.arg(project_ids))))
ORDER BY priority DESC, queued_at, id;

-- name: ListPlayQueuePeople :many
SELECT DISTINCT person_id FROM play_queue WHERE status = 'queued' AND not_before <= ?;

-- name: ListDuePlayQueue :many
SELECT * FROM play_queue WHERE status = 'queued' AND person_id = ? AND not_before <= ?
ORDER BY priority DESC, queued_at, id;

-- name: NextPlayQueueNotBefore :one
SELECT MIN(not_before) FROM play_queue WHERE status = 'queued' AND not_before > ?;

-- name: PlayQueueQueuedSince :one
SELECT EXISTS (
    SELECT 1 FROM play_queue
    WHERE target_type = ? AND target_id = ? AND queued_at >= ? AND auto_play_id = ? AND status IN ('queued', 'dispatching', 'started')
);

-- name: CountPlayQueueStarted :one
SELECT COUNT(*) AS started, MIN(q.decided_at) AS oldest FROM play_queue q
WHERE q.target_type = sqlc.arg(target_type) AND q.target_id = sqlc.arg(target_id) AND q.status = 'started'
  AND q.decided_at >= MAX(sqlc.arg(since), COALESCE(
      (SELECT resumed_at FROM play_queue_resumes r WHERE r.target_type = sqlc.arg(target_type) AND r.target_id = sqlc.arg(target_id)), 0));

-- name: UpsertPlayQueueResume :exec
INSERT INTO play_queue_resumes (target_type, target_id, resumed_by, resumed_at) VALUES (?, ?, ?, ?)
ON CONFLICT(target_type, target_id) DO UPDATE SET resumed_by = excluded.resumed_by, resumed_at = excluded.resumed_at;

-- name: RecoverDispatchingPlayQueue :exec
UPDATE play_queue SET
    status = CASE WHEN EXISTS (SELECT 1 FROM play_trails t WHERE t.id = play_queue.trail_id) THEN 'started' ELSE 'queued' END,
    trail_id = CASE WHEN EXISTS (SELECT 1 FROM play_trails t WHERE t.id = play_queue.trail_id) THEN trail_id ELSE '' END,
    decided_at = CASE WHEN EXISTS (SELECT 1 FROM play_trails t WHERE t.id = play_queue.trail_id) THEN decided_at ELSE NULL END
WHERE status = 'dispatching';

-- name: CountActivePlayTrailsByStarter :one
SELECT COUNT(*) FROM play_trails WHERE starter_id = ? AND state IN ('starting', 'running', 'waiting');
