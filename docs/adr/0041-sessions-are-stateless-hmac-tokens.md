# Sessions are stateless HMAC tokens with no server-side revocation list

A session is an HMAC-signed token (24h TTL) keyed by
`NEXUL_AUTH_SECRET`; logout discards it client-side. There is no session
table, so nothing to read on every request against a single-writer SQLite
file, and no session store to keep in step with the token.

ADR 0061 changes the revocation consequence without adding a session table.
Auth middleware reloads the User on every request and rejects disabled or
removed Accounts, so instance access changes take effect immediately.
Workspace Permissions are checked by each use-case as before. Rotating the
secret remains the escape hatch that invalidates every session at once.
