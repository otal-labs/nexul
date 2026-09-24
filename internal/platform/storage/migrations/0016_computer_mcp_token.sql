-- A personal access token minted for one paired computer's MCP connection; empty for a token the user named themselves.
ALTER TABLE personal_access_tokens ADD COLUMN computer_id TEXT NOT NULL DEFAULT '';

-- One active token per computer; also serves GetActiveComputerPAT.
CREATE UNIQUE INDEX IF NOT EXISTS idx_pats_active_computer
    ON personal_access_tokens(user_id, computer_id) WHERE computer_id != '' AND revoked_at IS NULL;
