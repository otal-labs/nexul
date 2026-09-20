# A fresh install requires no environment variables

`docker compose up` on a clean checkout has to reach a working setup wizard
with no `.env` authored by hand, because every required variable is a place an
install can fail before there is any UI to explain why. So nothing is
required: an unset `NEXUL_AUTH_SECRET` is generated on first start and
persisted next to the database, the install scripts generate the log store's
credentials, and the GitHub OAuth client id/secret moved out of the
environment into encrypted DB settings collected by the bootstrap screen.
Every externally-visible URL — OAuth callbacks, the MCP endpoint, runner
connection tokens — derives from the one instance URL the wizard asks for, so
there is no second place to keep in sync. TLS stays in front of the stack and
is never the server's concern.

Decided: 2026-09-11

## Consequences

Adding a new required env var is a regression, not a feature. A value the
product needs either has a working default, is generated on first start, or is
collected by the wizard and stored in the database — where changing it takes
effect live instead of requiring a restart.
