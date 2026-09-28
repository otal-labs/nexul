# 15 — Stored per-device sessions

**Type:** implementation
**Status:** done
**Blocked by:** None — can start immediately
**Decided in:** ticket 02

## What to build

Replace the stateless 24-hour HMAC session token with a stored session per
device. A `sessions` table (user, token hash, client `browser|desktop|phone`,
platform, label, IP, created, last active, expires) beside personal access
tokens, sharing their generate/hash/lookup helpers; tokens carry a `ses_`
prefix. OAuth sign-in, invitation acceptance and dev login all create a
session from the request (user agent parsed with a few lines of matching for
the common OS and browser names; an Electron user agent is "Nexul desktop").
The per-request auth reload loads the session with the user and rejects a
missing or expired row. Sliding expiry: 30 days since last use for browser
and desktop, 90 for phone; last active, IP and expiry are written at most
once an hour per session. Expired rows for a user are deleted at that user's
next sign-in. WebSocket auth (`?token=`) goes through the same path.

HTTP, own sessions only, no permission bit: list (current flagged), sign one
out, sign out everywhere else, and sign out the current session, which the
web app's Logout now calls before clearing local state.

Events `session.created` and `session.revoked` (catalog rows, outbox writes,
never carrying the token), pushed live to the owning user.

Write the ADR "Sessions are stored per device" superseding ADR 0041 and
amending the consequence in ADR 0061; it records why there is no MCP tool.
Add Session and Device to `CONTEXT.md`.

## Acceptance criteria

- [ ] Signing in by each of the three paths creates a session row; the old HMAC token no longer authenticates
- [ ] Signing a session out makes that token's next HTTP request and WebSocket dial fail
- [ ] Sign out everywhere else leaves only the current session; personal access tokens are untouched
- [ ] Last active is written at most once an hour per session (test with an injected clock)
- [ ] An expired session is rejected and is deleted at the user's next sign-in
- [ ] Web Logout deletes the session server-side

## Surfaces

- HTTP gateway routes for list / sign out one / sign out others / sign out current
- Events with catalog rows and outbox writes; live WebSocket push to the user
- No MCP tool, recorded in the ADR
- Docs: new ADR, ADR 0041 marked superseded, `CONTEXT.md`

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`,
`practices/react-guide.md` (for the Logout change), ADR 0041, ADR 0061, and
tickets 02 and 06.

## Verification

`go test ./...`, `make lint`, `make coverage`, `make sqlc-check`, and in
`web/`: `bun run lint`, `bun run typecheck`, `bun run test`.

## Files likely touched

- `internal/auth/` (service, middleware, handler, events, a new sessions file)
- `internal/platform/storage/migrations/`, `internal/platform/storage/queries/sessions.sql`
- `web/src/stores/sessionStore.ts`, `web/src/components/AccountMenuActions.tsx`, `web/src/hooks/AuthHooks.tsx`
- `docs/adr/`, `CONTEXT.md`

**Size:** L
