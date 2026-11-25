-- name: CreateOAuthClient :exec
INSERT INTO oauth_clients (id, name, redirect_uris, created_at) VALUES (?, ?, ?, ?);

-- name: GetOAuthClient :one
SELECT * FROM oauth_clients WHERE id = ?;

-- name: DeleteUngrantedOAuthClients :exec
DELETE FROM oauth_clients
WHERE oauth_clients.created_at < ?
  AND NOT EXISTS (SELECT 1 FROM personal_access_tokens p WHERE p.oauth_client_id = oauth_clients.id);
