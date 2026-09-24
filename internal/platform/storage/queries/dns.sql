-- name: UpsertServiceHostname :exec
INSERT INTO dns_service_hostnames (service, hostname, zone_id, zone, record_id, type, content, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(service) DO UPDATE SET
  hostname = excluded.hostname, zone_id = excluded.zone_id, zone = excluded.zone,
  record_id = excluded.record_id, type = excluded.type, content = excluded.content,
  created_at = excluded.created_at;

-- name: GetServiceHostname :one
SELECT * FROM dns_service_hostnames WHERE service = ?;

-- name: ListServiceHostnames :many
SELECT * FROM dns_service_hostnames ORDER BY created_at;

-- name: DeleteServiceHostname :execrows
DELETE FROM dns_service_hostnames WHERE service = ?;

-- name: SaveTunnel :exec
INSERT INTO dns_tunnels (id, name, account_id, status, hostname, zone_id, zone, record_id, service, agent_service_id, token, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name, account_id = excluded.account_id, status = excluded.status,
  hostname = excluded.hostname, zone_id = excluded.zone_id, zone = excluded.zone,
  record_id = excluded.record_id, service = excluded.service,
  agent_service_id = excluded.agent_service_id, token = excluded.token,
  updated_at = excluded.updated_at;

-- name: GetTunnel :one
SELECT * FROM dns_tunnels WHERE id = ?;

-- name: ListTunnels :many
SELECT * FROM dns_tunnels ORDER BY created_at;

-- name: DeleteTunnel :execrows
DELETE FROM dns_tunnels WHERE id = ?;

-- name: SaveGateway :exec
INSERT INTO dns_gateways (id, kind, docker_network, machine, networks, service_id, service_name, tunnel_id, zone_id, zone, server_address, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  kind = excluded.kind, docker_network = excluded.docker_network, machine = excluded.machine, networks = excluded.networks,
  service_id = excluded.service_id, service_name = excluded.service_name,
  tunnel_id = excluded.tunnel_id, zone_id = excluded.zone_id, zone = excluded.zone,
  server_address = excluded.server_address, updated_at = excluded.updated_at;

-- name: GetGateway :one
SELECT * FROM dns_gateways WHERE id = ?;

-- name: GetGatewayByNetwork :one
SELECT * FROM dns_gateways WHERE docker_network = ?;

-- name: ListGateways :many
SELECT * FROM dns_gateways ORDER BY created_at;

-- name: ListGatewaysByMachine :many
SELECT * FROM dns_gateways WHERE machine = ? ORDER BY created_at;

-- name: DeleteGateway :execrows
DELETE FROM dns_gateways WHERE id = ?;

-- name: SaveExposure :exec
INSERT INTO dns_exposures (id, gateway_id, hostname, service, service_id, port, zone_id, zone, record_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  gateway_id = excluded.gateway_id, hostname = excluded.hostname, service = excluded.service, service_id = excluded.service_id,
  port = excluded.port, zone_id = excluded.zone_id, zone = excluded.zone,
  record_id = excluded.record_id, updated_at = excluded.updated_at;

-- name: GetExposure :one
SELECT * FROM dns_exposures WHERE id = ?;

-- name: ListExposures :many
SELECT * FROM dns_exposures ORDER BY created_at;

-- name: ListExposuresByGateway :many
SELECT * FROM dns_exposures WHERE gateway_id = ? ORDER BY created_at;

-- name: ListExposuresByService :many
SELECT * FROM dns_exposures WHERE service_id = ? ORDER BY created_at;

-- name: ListExposuresByServiceName :many
SELECT * FROM dns_exposures WHERE service = ? ORDER BY created_at;

-- name: DeleteExposure :execrows
DELETE FROM dns_exposures WHERE id = ?;

-- name: SaveAccessServiceToken :exec
INSERT INTO dns_access_service_token (id, token_id, client_id, client_secret, created_at, updated_at)
VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  token_id = excluded.token_id, client_id = excluded.client_id,
  client_secret = excluded.client_secret, updated_at = excluded.updated_at;

-- name: GetAccessServiceToken :one
SELECT * FROM dns_access_service_token WHERE id = 1;

-- name: DeleteAccessServiceToken :execrows
DELETE FROM dns_access_service_token WHERE id = 1;
