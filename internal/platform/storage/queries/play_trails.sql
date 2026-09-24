-- name: CreatePlayTrail :exec
INSERT INTO play_trails (id, workspace_id, play_id, play_label, target_type, target_id, project_id, conversation_id, starter_id, via, selected_memory_ids, custom_instructions, move_to_status_id, harness_session_id, state, started_at, ended_at, last_error, reply_message_id, activity, computer_id, provider, model, question, failure_reason)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

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
