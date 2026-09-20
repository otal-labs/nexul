-- name: CreateStack :exec
INSERT INTO stacks (id, project_id, name, slug, machine, strategy, compose_path,
    env, docker_network, ports, mounts, command,
    build_repo_owner, build_repo_name, build_branch, build_dockerfile, build_compose_path,
    branch_deploy_rules, derived_from, branch, managed,
    created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetStack :one
SELECT * FROM stacks WHERE id = ?;

-- name: GetStackBySlugAndMachine :one
SELECT * FROM stacks WHERE slug = ? AND machine = ?;

-- name: GetStackByName :one
SELECT * FROM stacks WHERE name = ? ORDER BY created_at LIMIT 1;

-- name: ListStacksByProject :many
SELECT * FROM stacks WHERE (sqlc.arg(project_id) = '' OR project_id = sqlc.arg(project_id)) ORDER BY name;

-- name: ListStacksByBuildRepo :many
SELECT * FROM stacks WHERE derived_from = '' AND build_repo_owner = ? AND build_repo_name = ? ORDER BY name;

-- name: ListStacksByDerivedFrom :many
SELECT * FROM stacks WHERE derived_from = ? ORDER BY created_at;

-- name: UpdateStack :execrows
UPDATE stacks SET project_id = ?, name = ?, slug = ?, machine = ?, strategy = ?, compose_path = ?,
    env = ?, docker_network = ?, ports = ?, mounts = ?, command = ?,
    build_repo_owner = ?, build_repo_name = ?, build_branch = ?, build_dockerfile = ?, build_compose_path = ?,
    branch_deploy_rules = ?, derived_from = ?, branch = ?, managed = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeleteStack :execrows
DELETE FROM stacks WHERE id = ?;
