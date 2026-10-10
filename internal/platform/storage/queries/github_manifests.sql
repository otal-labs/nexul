-- name: DeleteExpiredGitHubManifests :exec
DELETE FROM github_manifest_requests WHERE expires_at <= ?;

-- name: StartGitHubManifest :exec
INSERT INTO github_manifest_requests (state_hash, initiator_hash, expires_at)
VALUES (?, ?, ?)
ON CONFLICT (initiator_hash) DO UPDATE SET state_hash = excluded.state_hash, expires_at = excluded.expires_at;

-- name: ConsumeGitHubManifest :execrows
DELETE FROM github_manifest_requests WHERE state_hash = ? AND initiator_hash = ? AND expires_at > ?;
