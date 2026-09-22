-- name: CreateDeploy :exec
INSERT INTO deploys (id, kind, stack_id, service_id, service, target, image, status, strategy,
    triggered_by, rule_id, rule_name, ticket_id, pr_number, created_at, updated_at, address)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDeploy :one
SELECT * FROM deploys WHERE id = ?;

-- name: ListDeploys :many
SELECT * FROM deploys ORDER BY created_at;

-- name: ListDeploysByService :many
SELECT * FROM deploys WHERE service = ? ORDER BY created_at DESC;

-- name: ListDeploysByStackID :many
SELECT * FROM deploys WHERE stack_id = ? ORDER BY created_at DESC;

-- name: ListDeploysByStatus :many
SELECT * FROM deploys WHERE status = ? ORDER BY created_at DESC;

-- name: CountActiveDeploys :one
SELECT COUNT(*) FROM deploys WHERE stack_id = ? AND status IN ('pending', 'running');

-- name: LastHealthyDeploy :one
SELECT * FROM deploys WHERE stack_id = ? AND status = 'healthy' ORDER BY created_at DESC LIMIT 1;

-- name: UpdateDeployStatus :execrows
UPDATE deploys SET status = ?, updated_at = ? WHERE id = ?;

-- name: SetDeployAddress :execrows
UPDATE deploys SET address = ?, updated_at = ? WHERE id = ?;
