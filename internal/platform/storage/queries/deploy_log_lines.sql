-- name: InsertDeployLogLine :exec
INSERT INTO deploy_log_lines (deploy_id, ts, phase, line) VALUES (?, ?, ?, ?);

-- name: ListDeployLogLines :many
SELECT seq, ts, phase, line FROM deploy_log_lines WHERE deploy_id = ? ORDER BY ts, seq;

-- name: ListDeployLogTail :many
SELECT seq, ts, phase, line FROM deploy_log_lines WHERE deploy_id = ? ORDER BY ts DESC, seq DESC LIMIT ?;

-- name: CountDeployLogLines :one
SELECT COUNT(*) FROM deploy_log_lines WHERE deploy_id = ?;
