-- name: CreatePlayTrail :one
-- The run records the play's newest instructions version in the same statement that inserts it.
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, conversation_id, starter_id, via, selected_memory_ids, custom_instructions, harness_session_id, state, started_at, ended_at, last_error, reply_message_id, activity, computer_id, provider, model, model_options, question, failure_reason, harness_project_id, worktree, passed_over, instructions_version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    (SELECT MAX(v.version) FROM play_instruction_versions v WHERE v.play_id = sqlc.arg(version_play_id)))
RETURNING instructions_version;

-- name: GetPlayTrail :one
SELECT * FROM play_trails WHERE id = ?;

-- name: UpdatePlayTrail :execrows
UPDATE play_trails SET conversation_id = ?, harness_session_id = ?, state = ?, ended_at = ?, last_error = ?, reply_message_id = ?, activity = ?, question = ?
WHERE id = ?;

-- name: ListPlayTrailsByTarget :many
SELECT * FROM play_trails WHERE target_type = ? AND target_id = ? ORDER BY started_at DESC, id DESC;

-- name: LatestPlayTrailForChoices :one
SELECT * FROM play_trails WHERE starter_id = ? AND play_id = ? AND project_id = ? ORDER BY started_at DESC, id DESC LIMIT 1;

-- name: ListActivePlayTrailsByTargets :many
SELECT * FROM play_trails WHERE target_type = ? AND target_id IN (sqlc.slice('ids')) AND state IN ('starting', 'running', 'waiting');

-- name: LatestPlayTrailInConversation :one
SELECT * FROM play_trails WHERE conversation_id = ? ORDER BY started_at DESC, id DESC LIMIT 1;

-- name: ListRunningPlayTrails :many
SELECT * FROM play_trails WHERE state IN ('starting', 'running');

-- name: ListPlayRunsByIDs :many
SELECT sqlc.embed(play_trails),
       CAST(COALESCE(tp.prefix || '-' || t.number, '') AS TEXT) AS target_key,
       COALESCE(t.title, d.title, s.name, ip.name, '') AS target_title,
       CAST(COALESCE((SELECT q.auto_play_id FROM play_queue q WHERE q.trail_id = play_trails.id AND q.trail_id <> '' LIMIT 1), '') AS TEXT) AS auto_play_id
FROM play_trails
LEFT JOIN tickets t ON play_trails.target_type = 'ticket' AND t.id = play_trails.target_id
LEFT JOIN projects tp ON tp.id = t.project_id
LEFT JOIN docs d ON play_trails.target_type = 'doc' AND d.id = play_trails.target_id
LEFT JOIN stacks s ON play_trails.target_type = 'stack' AND s.id = play_trails.target_id
LEFT JOIN projects ip ON play_trails.target_type = 'interview' AND ip.id = play_trails.target_id
WHERE play_trails.id IN (SELECT value FROM json_each(sqlc.arg(ids)));
