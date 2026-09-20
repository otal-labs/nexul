-- name: SavePairingComputer :exec
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  kind = excluded.kind, name = excluded.name, server_url = excluded.server_url, bearer_token = excluded.bearer_token,
  token_expires_at = excluded.token_expires_at, harness_version = excluded.harness_version, updated_at = excluded.updated_at;

-- name: GetPairingComputer :one
SELECT * FROM pairing_computers WHERE id = ? AND user_id = ?;

-- name: GetPairingComputerByID :one
SELECT * FROM pairing_computers WHERE id = ?;

-- name: ListPairingComputers :many
SELECT * FROM pairing_computers WHERE user_id = ? ORDER BY created_at DESC;

-- name: DeletePairingComputer :execrows
DELETE FROM pairing_computers WHERE id = ? AND user_id = ?;

-- name: GetPairingProjectLink :one
SELECT * FROM pairing_project_links WHERE project_id = ?;

-- name: SavePairingProjectLink :exec
INSERT INTO pairing_project_links (project_id, computer_id, harness_project_id, provider, model, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(project_id) DO UPDATE SET
  computer_id = excluded.computer_id, harness_project_id = excluded.harness_project_id,
  provider = excluded.provider, model = excluded.model, updated_at = excluded.updated_at;

-- name: DeletePairingProjectLink :exec
DELETE FROM pairing_project_links WHERE project_id = ?;

-- name: GetPairingDefaults :one
SELECT default_computer_id, fallback_project_id, provider, model FROM pairing_user_defaults WHERE user_id = ?;

-- name: SavePairingDefaults :exec
INSERT INTO pairing_user_defaults (user_id, default_computer_id, fallback_project_id, provider, model)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
  default_computer_id = excluded.default_computer_id, fallback_project_id = excluded.fallback_project_id,
  provider = excluded.provider, model = excluded.model;
