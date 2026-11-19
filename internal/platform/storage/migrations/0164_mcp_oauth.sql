-- An MCP client that registered itself (OAuth dynamic client registration); a public client, so it holds no secret.
CREATE TABLE IF NOT EXISTS oauth_clients (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    redirect_uris TEXT NOT NULL,
    created_at    INTEGER NOT NULL
);

-- A sign-in grant is a personal access token row: its access token expires and its refresh token mints the next one.
ALTER TABLE personal_access_tokens ADD COLUMN oauth_client_id TEXT NOT NULL DEFAULT '';
ALTER TABLE personal_access_tokens ADD COLUMN refresh_hash TEXT;
ALTER TABLE personal_access_tokens ADD COLUMN expires_at INTEGER;
-- Serves RotatePATByRefresh, the token endpoint's refresh lookup.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pats_refresh_hash ON personal_access_tokens(refresh_hash) WHERE refresh_hash IS NOT NULL;
