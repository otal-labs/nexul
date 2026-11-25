-- name: InsertShellJob :exec
INSERT INTO runner_shell_jobs (id, computer_id, owner_id, requester_id, adapter, shell, command, cwd, timeout_seconds, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: StartShellJob :execrows
UPDATE runner_shell_jobs SET status = 'running', started_at = ? WHERE id = ? AND status = 'queued';

-- name: FinishShellJob :execrows
UPDATE runner_shell_jobs SET status = ?, exit_code = ?, error = ?, output = ?, output_truncated = ?, ended_at = ?
WHERE id = ? AND status IN ('queued', 'running');

-- name: GetShellJob :one
SELECT * FROM runner_shell_jobs WHERE id = ?;

-- name: ListShellJobs :many
-- A computer's jobs newest first; a requester id narrows them to that person's own.
SELECT * FROM runner_shell_jobs
WHERE computer_id = sqlc.arg(computer_id) AND (sqlc.arg(requester_id) = '' OR requester_id = sqlc.arg(requester_id))
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountShellJobs :one
SELECT COUNT(*) FROM runner_shell_jobs
WHERE computer_id = sqlc.arg(computer_id) AND (sqlc.arg(requester_id) = '' OR requester_id = sqlc.arg(requester_id));

-- name: PurgeShellJobOutput :execrows
-- Empties the output of up to lim jobs that ended before the cutoff, keeping the rest of each record.
UPDATE runner_shell_jobs SET output = '', output_purged = 1
WHERE id IN (SELECT j.id FROM runner_shell_jobs j WHERE j.output != '' AND j.ended_at < sqlc.arg(before) LIMIT sqlc.arg(lim));

-- name: FailUnfinishedShellJobs :execrows
-- A job still queued or running when the server starts lost its runner's answer with the server that waited for it.
UPDATE runner_shell_jobs SET status = 'failed', error = 'the server restarted before the command ended', ended_at = ?
WHERE status IN ('queued', 'running');

-- name: SetPairingComputerShell :execrows
UPDATE pairing_computers SET shell_enabled = ?, updated_at = ? WHERE id = ? AND user_id = ?;
