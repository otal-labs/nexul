-- name: GetConnectorAppConfig :one
SELECT * FROM connector_app_config WHERE connector_id = ?;

-- name: SetConnectorAppConfig :exec
INSERT INTO connector_app_config (connector_id, client_id, client_secret, base_url, app_slug)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(connector_id) DO UPDATE SET
  client_id = excluded.client_id, client_secret = excluded.client_secret,
  base_url = excluded.base_url, app_slug = excluded.app_slug;
