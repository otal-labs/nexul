-- The refresh token each single sign-on person's latest sign-in returned, sealed, and when the provider last confirmed it.
CREATE TABLE IF NOT EXISTS sso_grants (
    user_id       TEXT    PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    refresh_token BLOB    NOT NULL,
    checked_at    INTEGER NOT NULL
);

-- ListStaleSSOGrants: the oldest unchecked grants first.
CREATE INDEX IF NOT EXISTS idx_sso_grants_checked_at ON sso_grants (checked_at);
