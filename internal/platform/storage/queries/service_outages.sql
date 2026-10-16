-- name: CreateOutage :exec
INSERT INTO service_outages (id, stack_id, service, container, started_at, exit_code, oom, restart_count)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateOutage :exec
UPDATE service_outages SET container = ?, exit_code = ?, oom = ?, restart_count = ? WHERE id = ?;

-- name: CloseOutage :execrows
UPDATE service_outages SET ended_at = ? WHERE id = ? AND ended_at IS NULL;

-- name: GetOpenOutage :one
SELECT * FROM service_outages WHERE stack_id = ? AND service = ? AND ended_at IS NULL;

-- name: ListOpenOutagesByStack :many
SELECT * FROM service_outages WHERE stack_id = ? AND ended_at IS NULL ORDER BY started_at, id;

-- name: LatestOutageEnd :one
SELECT ended_at FROM service_outages WHERE stack_id = ? AND service = ? AND ended_at IS NOT NULL ORDER BY ended_at DESC LIMIT 1;
