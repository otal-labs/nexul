-- name: SavePairingComputer :exec
-- The tunnel columns and position are written on insert only, so re-pairing never drops a computer's tunnel or moves it.
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at,
  tunnel_id, tunnel_hostname, tunnel_zone_id, tunnel_record_id, tunnel_access_app_id, position)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  kind = excluded.kind, name = excluded.name, server_url = excluded.server_url, bearer_token = excluded.bearer_token,
  token_expires_at = excluded.token_expires_at, harness_version = excluded.harness_version, updated_at = excluded.updated_at;

-- name: GetPairingComputer :one
SELECT * FROM pairing_computers WHERE id = ? AND user_id = ?;

-- name: ListPairingComputers :many
SELECT * FROM pairing_computers WHERE user_id = ? ORDER BY position, created_at DESC;

-- name: NextPairingComputerPosition :one
-- The place after the last of a person's computers, where a new one goes.
SELECT CAST(COALESCE(MAX(position) + 1, 0) AS INTEGER) FROM pairing_computers WHERE user_id = ?;

-- name: SetPairingComputerPosition :execrows
UPDATE pairing_computers SET position = ? WHERE id = ? AND user_id = ?;

-- name: SetPairingComputerPlacement :execrows
UPDATE pairing_computers SET runs_at_once = ?, chat_project_id = ?, updated_at = ? WHERE id = ? AND user_id = ?;

-- name: CountPairingComputerRuns :many
-- The play runs going on each computer, for its runs at once (ADR 0166).
SELECT computer_id, COUNT(*) AS running FROM play_trails
WHERE computer_id IN (SELECT value FROM json_each(sqlc.arg(computer_ids))) AND state IN ('starting', 'running', 'waiting')
GROUP BY computer_id;

-- name: ListPairingTunnelComputers :many
SELECT * FROM pairing_computers WHERE tunnel_id != '' ORDER BY created_at, id;

-- name: SwitchPairingComputerKind :one
UPDATE pairing_computers SET kind = sqlc.arg(to_kind), harness_version = sqlc.arg(harness_version), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND kind = sqlc.arg(from_kind)
RETURNING user_id;

-- name: DeletePairingComputer :execrows
DELETE FROM pairing_computers WHERE id = ? AND user_id = ?;

-- name: GetPairingProjectLink :one
SELECT * FROM pairing_project_links WHERE user_id = ? AND project_id = ?;

-- name: ListPairingProjectLinks :many
SELECT * FROM pairing_project_links WHERE user_id = ? ORDER BY project_id;

-- name: SavePairingProjectLink :exec
INSERT INTO pairing_project_links (user_id, project_id, provider, model, model_options, start_in, placement, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, project_id) DO UPDATE SET
  provider = excluded.provider, model = excluded.model, model_options = excluded.model_options, start_in = excluded.start_in,
  placement = excluded.placement, updated_at = excluded.updated_at;

-- name: ListPairingProjectComputers :many
-- A person's computers for one project: those in its own order by position, then the ones only holding a checkout.
SELECT * FROM pairing_project_computers WHERE user_id = ? AND project_id = ? ORDER BY position IS NULL, position, computer_id;

-- name: ListPairingProjectComputersByUser :many
SELECT * FROM pairing_project_computers WHERE user_id = ? ORDER BY project_id, position IS NULL, position, computer_id;

-- name: ClearPairingProjectOrder :exec
UPDATE pairing_project_computers SET position = NULL WHERE user_id = ? AND project_id = ?;

-- name: SetPairingProjectComputerPosition :exec
INSERT INTO pairing_project_computers (user_id, project_id, computer_id, position) VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, project_id, computer_id) DO UPDATE SET position = excluded.position;

-- name: SetPairingProjectCheckout :exec
INSERT INTO pairing_project_computers (user_id, project_id, computer_id, harness_project_id) VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, project_id, computer_id) DO UPDATE SET harness_project_id = excluded.harness_project_id;

-- name: PrunePairingProjectComputers :exec
DELETE FROM pairing_project_computers WHERE user_id = ? AND project_id = ? AND position IS NULL AND harness_project_id = '';

-- name: DeletePairingProjectLink :exec
DELETE FROM pairing_project_links WHERE user_id = ? AND project_id = ?;

-- name: GetPairingDefaults :one
SELECT * FROM pairing_user_defaults WHERE user_id = ?;

-- name: SavePairingDefaults :exec
INSERT INTO pairing_user_defaults (user_id, provider, model, model_options, start_in, placement, clone_automatically)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
  provider = excluded.provider, model = excluded.model, model_options = excluded.model_options, start_in = excluded.start_in,
  placement = excluded.placement, clone_automatically = excluded.clone_automatically;

-- name: SetPairingComputerSetupConfirmedAt :execrows
UPDATE pairing_computers SET setup_confirmed_at = ? WHERE id = ? AND user_id = ?;

-- name: ListPairingProviderSetups :many
SELECT * FROM pairing_provider_setups WHERE computer_id = ? ORDER BY provider;

-- name: ListPairingComputersWithOutdatedSkills :many
-- A person's own computers with a confirmed provider whose skills predate the version this server ships, for the settings dot.
SELECT DISTINCT s.computer_id FROM pairing_provider_setups s
JOIN pairing_computers c ON c.id = s.computer_id
WHERE c.user_id = ? AND s.confirmed_at IS NOT NULL AND s.skills_version <> ?;

-- name: SavePairingProviderSetup :exec
INSERT INTO pairing_provider_setups (computer_id, provider, confirmed_at, skills_json, skills_version, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(computer_id, provider) DO UPDATE SET
  confirmed_at = excluded.confirmed_at, skills_json = excluded.skills_json, skills_version = excluded.skills_version,
  updated_at = excluded.updated_at;

-- name: DeletePairingProviderSetup :exec
DELETE FROM pairing_provider_setups WHERE computer_id = ? AND provider = ?;

-- name: DeletePairingSetupTurnsForProvider :exec
DELETE FROM pairing_setup_turns WHERE computer_id = ? AND provider = ?;

-- name: SetPairingComputerSetupMCPToken :execrows
UPDATE pairing_computers SET setup_mcp_token = ? WHERE id = ? AND user_id = ?;

-- name: SavePairingSetupTurn :exec
INSERT INTO pairing_setup_turns (id, run_id, computer_id, provider, provider_name, model, kind, state, status, transcript, started_at, updated_at, ended_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  state = excluded.state, status = excluded.status, transcript = excluded.transcript, updated_at = excluded.updated_at,
  ended_at = excluded.ended_at;

-- name: ListPairingSetupTurnsLatest :many
-- The newest turn of each provider on a computer, whichever run it belongs to, so a retry sits beside the rest.
SELECT t.id, t.run_id, t.provider, t.provider_name, t.model, t.kind, t.state, t.status, t.updated_at FROM pairing_setup_turns t
WHERE t.computer_id = sqlc.arg(computer_id) AND t.id = (
  SELECT l.id FROM pairing_setup_turns l WHERE l.computer_id = sqlc.arg(computer_id) AND l.provider = t.provider
  ORDER BY l.started_at DESC, l.id DESC LIMIT 1
)
ORDER BY t.provider;

-- name: SetPairingComputerSetupChoices :execrows
UPDATE pairing_computers SET setup_skipped_providers = ?, setup_models = ?, setup_model_options = ?, setup_folder = ? WHERE id = ? AND user_id = ?;

-- name: SetPairingComputerFacts :execrows
UPDATE pairing_computers SET facts = ?, facts_at = ? WHERE id = ? AND user_id = ?;

-- name: ClearPairingComputerTunnel :execrows
UPDATE pairing_computers SET tunnel_id = '', tunnel_hostname = '', tunnel_zone_id = '', tunnel_record_id = '', tunnel_access_app_id = '',
  updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: ListPairingComputersToMove :many
-- Every computer still on a tunnel or an address whose owner has no move notice yet, oldest first per owner.
SELECT c.id, c.user_id FROM pairing_computers c
WHERE (c.tunnel_id != '' OR (c.server_url != '' AND c.server_url NOT LIKE sqlc.arg(runner_address_pattern)))
  AND NOT EXISTS (SELECT 1 FROM pairing_move_notices n WHERE n.user_id = c.user_id)
ORDER BY c.user_id, c.created_at, c.id;

-- name: InsertPairingMoveNotice :execrows
INSERT INTO pairing_move_notices (user_id, notified_at) VALUES (?, ?) ON CONFLICT(user_id) DO NOTHING;

-- name: ListPairingComputersFor :many
-- A person's own computers, then those a live rule naming them shares with See (ADR 0148), each with its owner's name.
SELECT sqlc.embed(c), CAST(coalesce(po.allow, '') AS TEXT) AS rule_allow,
    CAST(coalesce(nullif(o.display_name, ''), nullif(o.name, ''), o.login) AS TEXT) AS owner_name
FROM pairing_computers c
JOIN users o ON o.id = c.user_id
LEFT JOIN permission_overwrites po ON po.resource_type = 'computer' AND po.resource_id = c.id AND po.target = 'person'
    AND po.user_id = sqlc.arg(user_id)
WHERE c.user_id = sqlc.arg(user_id)
   OR (EXISTS (SELECT 1 FROM json_each(po.allow) WHERE value = 'computers:see')
       AND o.account_status = 'active'
       AND EXISTS (SELECT 1 FROM users me WHERE me.id = sqlc.arg(user_id) AND me.account_status = 'active')
       AND EXISTS (SELECT 1 FROM workspace_members a JOIN workspace_members b ON b.workspace_id = a.workspace_id
                   WHERE a.user_id = sqlc.arg(user_id) AND b.user_id = c.user_id))
ORDER BY po.allow IS NOT NULL, c.position, c.created_at DESC;

-- name: ListUsersWithReadyComputer :many
SELECT DISTINCT user_id FROM pairing_computers
WHERE token_expires_at > 0 AND setup_confirmed_at IS NOT NULL AND user_id IN (SELECT value FROM json_each(sqlc.arg(user_ids)));
