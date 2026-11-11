-- The instance's one single sign-on provider (OIDC): a singleton row, kept when the license key lapses so a renewed key
-- turns sign-in back on without being set up again. The client secret is sealed, or a secret reference as typed.
CREATE TABLE IF NOT EXISTS sso_settings (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    issuer_url    TEXT    NOT NULL,
    client_id     TEXT    NOT NULL,
    client_secret TEXT    NOT NULL,
    display_name  TEXT    NOT NULL,
    join_role_id  TEXT    NOT NULL,
    sso_only      INTEGER NOT NULL DEFAULT 0,
    updated_by    TEXT    NOT NULL DEFAULT '',
    updated_at    INTEGER NOT NULL
);

-- FindUserByEmailIdentity: an SSO sign-in's verified email against the accounts whose Google or Discord login is one.
CREATE INDEX IF NOT EXISTS idx_user_identities_login_nocase ON user_identities (provider, login COLLATE NOCASE);
