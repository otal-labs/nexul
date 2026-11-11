-- name: GetLicenseKey :one
SELECT key FROM license_key WHERE id = 1;

-- name: SetLicenseKey :exec
INSERT INTO license_key (id, key, set_by, set_at) VALUES (1, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET key = excluded.key, set_by = excluded.set_by, set_at = excluded.set_at;

-- name: DeleteLicenseKey :exec
DELETE FROM license_key WHERE id = 1;
