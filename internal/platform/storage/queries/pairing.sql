-- name: SavePairingComputer :exec
-- The tunnel columns are written on insert only, so re-pairing never drops a computer's tunnel.
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at,
  tunnel_id, tunnel_hostname, tunnel_zone_id, tunnel_record_id, tunnel_access_app_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  kind = excluded.kind, name = excluded.name, server_url = excluded.server_url, bearer_token = excluded.bearer_token,
  token_expires_at = excluded.token_expires_at, harness_version = excluded.harness_version, updated_at = excluded.updated_at;

-- name: GetPairingComputer :one
SELECT * FROM pairing_computers WHERE id = ? AND user_id = ?;

-- name: GetPairingComputerByID :one
SELECT * FROM pairing_computers WHERE id = ?;

-- name: ListPairingComputers :many
SELECT * FROM pairing_computers WHERE user_id = ? ORDER BY created_at DESC;

-- name: PairingComputerTunnelHostnameExists :one
SELECT EXISTS (SELECT 1 FROM pairing_computers WHERE tunnel_hostname = ? AND tunnel_hostname != '');

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

-- name: SetPairingComputerSetupConfirmedAt :execrows
UPDATE pairing_computers SET setup_confirmed_at = ? WHERE id = ? AND user_id = ?;

-- name: ListPairingProviderSetups :many
SELECT * FROM pairing_provider_setups WHERE computer_id = ? ORDER BY provider;

-- name: SavePairingProviderSetup :exec
INSERT INTO pairing_provider_setups (computer_id, provider, confirmed_at, skills_json, updated_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(computer_id, provider) DO UPDATE SET
  confirmed_at = excluded.confirmed_at, skills_json = excluded.skills_json, updated_at = excluded.updated_at;

-- name: SetPairingComputerSetupMCPToken :execrows
UPDATE pairing_computers SET setup_mcp_token = ? WHERE id = ? AND user_id = ?;

-- name: SavePairingSetupTurn :exec
INSERT INTO pairing_setup_turns (id, run_id, computer_id, provider, provider_name, model, state, status, transcript, started_at, updated_at, ended_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  state = excluded.state, status = excluded.status, transcript = excluded.transcript, updated_at = excluded.updated_at,
  ended_at = excluded.ended_at;

-- name: ListPairingSetupTurnsLatest :many
-- The newest turn of each provider on a computer, whichever run it belongs to, so a retry sits beside the rest.
SELECT t.id, t.run_id, t.provider, t.provider_name, t.model, t.state, t.status, t.updated_at FROM pairing_setup_turns t
WHERE t.computer_id = sqlc.arg(computer_id) AND t.id = (
  SELECT l.id FROM pairing_setup_turns l WHERE l.computer_id = sqlc.arg(computer_id) AND l.provider = t.provider
  ORDER BY l.started_at DESC, l.id DESC LIMIT 1
)
ORDER BY t.provider;
