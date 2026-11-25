-- The refresh token a grant last rotated away from, so presenting it again revokes the grant; grants rotated before this hold none.
ALTER TABLE personal_access_tokens ADD COLUMN previous_refresh_hash TEXT;
-- Serves GetPATByPreviousRefresh, the token endpoint's reuse check.
CREATE INDEX IF NOT EXISTS idx_pats_previous_refresh_hash ON personal_access_tokens(previous_refresh_hash) WHERE previous_refresh_hash IS NOT NULL;
