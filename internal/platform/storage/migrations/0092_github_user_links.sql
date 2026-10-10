-- A person's own GitHub App user token (ADR 0147): what lists the repositories they can open where the App is
-- installed. Encrypted at rest like a connector's; arrives on the next GitHub sign-in or Connect GitHub, so there is
-- nothing to backfill. A refresh GitHub refused leaves needs_reconnect set and the tokens empty.
CREATE TABLE IF NOT EXISTS github_user_links (
    user_id            TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_token       TEXT NOT NULL,
    refresh_token      TEXT NOT NULL DEFAULT '',
    expires_at         INTEGER NOT NULL DEFAULT 0,
    refresh_expires_at INTEGER NOT NULL DEFAULT 0,
    needs_reconnect    INTEGER NOT NULL DEFAULT 0,
    connected_at       INTEGER NOT NULL
);

-- An install link no longer assigns its installation to a workspace (ADR 0147): attaching a repository does.
DROP TABLE IF EXISTS github_install_states;
