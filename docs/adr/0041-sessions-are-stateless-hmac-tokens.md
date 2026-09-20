# Sessions are stateless HMAC tokens with no server-side revocation list

A session is an HMAC-signed token (24h TTL) keyed by
`NEXUL_AUTH_SECRET`; logout discards it client-side. There is no session
table, so nothing to read on every request against a single-writer SQLite
file, and no session store to keep in step with the token.

The accepted consequence is that revocation is not immediate: removing someone
from the allowlist, or changing their permissions, takes effect within the
token's TTL. The escape hatch is rotating the secret, which invalidates every
session at once.
