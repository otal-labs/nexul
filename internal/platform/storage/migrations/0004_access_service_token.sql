-- The instance's one Cloudflare Access service token; client_secret is encrypted at rest.
CREATE TABLE IF NOT EXISTS dns_access_service_token (
    id             INTEGER PRIMARY KEY CHECK (id = 1),
    token_id       TEXT NOT NULL,
    client_id      TEXT NOT NULL,
    client_secret  TEXT NOT NULL,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
);
