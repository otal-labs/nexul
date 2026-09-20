-- name: CreateContainer :exec
INSERT INTO services (id, stack_id, name, declared, container_name, image, status, networks, ports, observed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpsertContainer :exec
INSERT INTO services (id, stack_id, name, declared, container_name, image, status, networks, ports, observed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (stack_id, name) DO UPDATE SET
    declared = excluded.declared,
    container_name = excluded.container_name,
    image = excluded.image,
    status = excluded.status,
    networks = excluded.networks,
    ports = excluded.ports,
    observed_at = excluded.observed_at;

-- name: GetContainer :one
SELECT * FROM services WHERE id = ?;

-- name: ListContainersByStack :many
SELECT * FROM services WHERE stack_id = ? ORDER BY name;

-- name: DeleteContainersByStack :exec
DELETE FROM services WHERE stack_id = ?;

-- name: ListContainerNamesByMachine :many
SELECT c.container_name FROM services c JOIN stacks s ON s.id = c.stack_id
WHERE s.machine = ? AND c.container_name <> '' ORDER BY c.container_name;
