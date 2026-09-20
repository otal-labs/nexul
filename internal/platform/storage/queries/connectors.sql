-- name: GetConnectorCredentials :one
SELECT * FROM connector_credentials WHERE connector_id = ?;

-- name: SaveConnectorCredentials :exec
INSERT INTO connector_credentials (connector_id, access_token, refresh_token, expires_at, connected_by, connected_at, manual_fields)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(connector_id) DO UPDATE SET
  access_token = excluded.access_token, refresh_token = excluded.refresh_token,
  expires_at = excluded.expires_at, connected_by = excluded.connected_by,
  connected_at = excluded.connected_at, manual_fields = excluded.manual_fields;

-- name: DeleteConnectorCredentials :exec
DELETE FROM connector_credentials WHERE connector_id = ?;
