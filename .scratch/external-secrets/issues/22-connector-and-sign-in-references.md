# 22 — Connector and sign-in secrets resolve by reference, survive an outage, and re-resolve on 401

**Status:** resolved

**Blocked by:** 11, 21

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ADR 0181, the spec (The server).

## What to build

- Connector secret fields and app config secrets that are references resolve at server start and on
  save (inside the existing live verification), then stay in memory.
- 1Password unreachable at start: the server starts; the connector or sign-in button shows
  "1Password unreachable" and retries with backoff capped at about 100 reads a day.
- A 401/403 from GitHub, Cloudflare or LiveKit re-resolves that credential once, at most every
  5 minutes, and retries the call.

## Acceptance criteria

- [ ] With the fake: an unreachable resolver at start leaves every other connector and passkeys working.
- [ ] A 401 triggers one re-resolve and a retried call; a second 401 within 5 minutes does not re-resolve.
- [ ] The backoff's daily read count is asserted against a fake clock.

Verify: `make lint`, `go test ./internal/connectors/... ./internal/auth/...`, `make coverage`.

## Comments

Built: `secretref.Cache` (read once and kept in memory, failures back off to about 100 reads a day, `Refresh` at most every 5 minutes), the resolving layer at the connectors seam (`ResolvedAppConfigs`, `AccessToken`, `ManualCredentials`; the raw stores keep the reference), a save that verifies the resolved value, a `secret` state on the failing connector's status and card, a warm step after the managers open, 401/403 re-resolve in the Cloudflare, LiveKit and GitHub App clients, and sign-in client secrets resolved at `providerClient`. A sign-in provider whose secret can't be resolved fails that login with an error; the public sign-in page does not advertise the manager's state to anonymous visitors. Checked with real 1Password: a Cloudflare save through a reference reached Cloudflare with the resolved value (Cloudflare rejected the dummy token), a missing field was refused as `API token: field missing`, and neither the value nor the reference reached the log or the database.
