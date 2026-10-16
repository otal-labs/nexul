-- name: CreateErrorRateEpisode :exec
INSERT INTO error_rate_episodes (id, stack_id, started_at, peak_percent, requests, errors, limit_percent, window_minutes, min_requests)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateErrorRateEpisodePeak :exec
UPDATE error_rate_episodes SET peak_percent = ?, requests = ?, errors = ? WHERE id = ?;

-- name: CloseErrorRateEpisode :execrows
UPDATE error_rate_episodes SET ended_at = ? WHERE id = ? AND ended_at IS NULL;

-- name: GetOpenErrorRateEpisode :one
SELECT * FROM error_rate_episodes WHERE stack_id = ? AND ended_at IS NULL;

-- name: LatestErrorRateEnd :one
SELECT ended_at FROM error_rate_episodes WHERE stack_id = ? AND ended_at IS NOT NULL ORDER BY ended_at DESC LIMIT 1;

-- name: ListStacksWithErrorRate :many
SELECT * FROM stacks WHERE error_rate_enabled = 1 ORDER BY id;
