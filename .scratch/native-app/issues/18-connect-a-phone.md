# 18 — Connect a phone

**Type:** implementation
**Status:** done
**Blocked by:** 15, 17
**Decided in:** tickets 05, 06

## What to build

Server: a `connect_codes` table and two endpoints. Issuing (session
authentication only, personal access tokens refused) returns a fresh
12-character Crockford base32 code shown as `XXXX-XXXX-XXXX`, valid two
minutes, and invalidates the user's previous code. The public exchange takes
`{code, device: {model, os, app_version}}`, marks the code used, and returns a
phone session (ticket 15, label "Android · <model>") plus the server
version; every failure is one generic "invalid or expired" error, and five
failures from one IP in ten minutes return 429 (same limiter shape as
setup-code unlocking).

Web: the *Connect a phone* card beside the desktop card (side by side from
1024px). The QR code renders `nexul://connect?host=<instance URL>&code=<code>`
with `uqr` as an SVG on a white tile, plus the code as mono text, a mono
countdown, and New code. Expiry fades the QR to 10% with a New code button
over it. When `session.created` arrives for a phone while a code is showing,
the card plays the connected hero from ticket 05 (800ms crossfade to the
check tile, the new row enters with the 800ms rise and the 5600ms glow after
an 800ms hold) and stays confirmed until the page is left.

## Acceptance criteria

- [ ] A code works exactly once, never after two minutes, and never after a newer code was issued
- [ ] A personal access token cannot issue a code
- [ ] The sixth wrong code from one IP inside ten minutes gets 429
- [ ] Exchanging a code creates a phone session that appears on the web Devices tab live, with the hero motion
- [ ] The QR scans with a stock phone camera and yields the link
- [ ] Reduced motion leaves the confirmed state without movement

## Surfaces

- HTTP (issue, exchange); events (the session's `session.created`); live push
- No MCP tool (ticket 06)
- Docs: `CONTEXT.md` gains Connect code; the design-language motion baseline records the connected hero as its one exception

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/react-guide.md`,
`practices/design-language.md`, `practices/testing.md`, tickets 05 and 06.

## Verification

`go test ./...`, `make lint`, `make coverage`, `make sqlc-check`; in `web/`: `bun run lint`, `bun run typecheck`, `bun run test`; exchange a code with curl and watch the web card flip.

## Files likely touched

- `internal/auth/` (connect codes, handler, limiter)
- `internal/platform/storage/migrations/`, `internal/platform/storage/queries/connect_codes.sql`
- `web/package.json` (`uqr`), `web/src/components/you/`, `practices/design-language.md`, `CONTEXT.md`

**Size:** M
