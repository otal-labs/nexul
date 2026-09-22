-- name: InsertDeployLogLine :exec
INSERT INTO deploy_log_lines (deploy_id, ts, phase, line) VALUES (?, ?, ?, ?);

-- name: ListDeployLogLines :many
SELECT seq, ts, phase, line FROM deploy_log_lines WHERE deploy_id = ? ORDER BY ts, seq;
