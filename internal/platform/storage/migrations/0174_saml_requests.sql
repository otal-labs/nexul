-- The SAML sign-in requests Nexul sent and waits on (ADR 0189). The provider's response posts back cross-site without
-- the browser's cookies, so it is tied to its request by InResponseTo alone; using a row deletes it, refusing a replay.
CREATE TABLE IF NOT EXISTS saml_requests (
    id         TEXT    PRIMARY KEY,
    created_at INTEGER NOT NULL
);

-- DeleteExpiredSAMLRequests: the requests older than their 10 minutes.
CREATE INDEX IF NOT EXISTS idx_saml_requests_created_at ON saml_requests (created_at);
