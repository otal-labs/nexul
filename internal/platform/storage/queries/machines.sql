-- name: CreateMachine :exec
INSERT INTO machines (id, name, stack_root, reported_hostname, first_seen, last_seen) VALUES (?, ?, ?, ?, ?, ?);

-- name: GetMachine :one
SELECT * FROM machines WHERE id = ?;

-- name: GetMachineByName :one
SELECT * FROM machines WHERE name = ?;

-- name: ListMachines :many
SELECT * FROM machines ORDER BY name;

-- name: ListMachinesIn :many
SELECT * FROM machines WHERE id IN (SELECT value FROM json_each(sqlc.arg(ids))) ORDER BY name;

-- name: RenameMachine :execrows
UPDATE machines SET name = ? WHERE id = ?;

-- name: SetMachineStackRoot :execrows
UPDATE machines SET stack_root = ? WHERE id = ?;

-- name: TouchMachine :execrows
UPDATE machines SET last_seen = ?, reported_hostname = ? WHERE id = ?;

-- name: ListMachineStackNames :many
SELECT name FROM stacks WHERE machine = ? ORDER BY name;

-- name: ListMachineGatewayNetworks :many
SELECT docker_network FROM dns_gateways WHERE machine = ? ORDER BY created_at;

-- name: DeleteMachine :execrows
DELETE FROM machines WHERE id = ?;

-- name: RevokeMachineRunnerCredentials :exec
UPDATE runner_credentials SET revoked_at = ?
WHERE revoked_at IS NULL AND runner_id IN (SELECT id FROM runners WHERE machine_id = ? AND owner_user_id = '');

-- name: DeleteMachineRunners :exec
DELETE FROM runners WHERE machine_id = ? AND owner_user_id = '';

-- name: DeleteMachineEnrollments :exec
DELETE FROM runner_enrollment_codes WHERE machine = ? AND owner_user_id = '';
