-- name: CreateMachine :exec
INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetMachine :one
SELECT * FROM machines WHERE id = ?;

-- name: GetMachineByName :one
SELECT * FROM machines WHERE name = ?;

-- name: ListMachines :many
SELECT * FROM machines ORDER BY name;

-- name: RenameMachine :execrows
UPDATE machines SET name = ? WHERE id = ?;

-- name: SetMachineStackRoot :execrows
UPDATE machines SET stack_root = ? WHERE id = ?;

-- name: TouchMachine :execrows
UPDATE machines SET last_seen = ?, reported_hostname = ? WHERE id = ?;
