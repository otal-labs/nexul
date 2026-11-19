-- name: CreateOAuthClient :exec
INSERT INTO oauth_clients (id, name, redirect_uris, created_at) VALUES (?, ?, ?, ?);

-- name: GetOAuthClient :one
SELECT * FROM oauth_clients WHERE id = ?;
