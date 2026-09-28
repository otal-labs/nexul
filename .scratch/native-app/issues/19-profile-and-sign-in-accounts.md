# 19 — Profile and linked sign-in accounts

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 16
**Decided in:** tickets 03, 05

## What to build

Move sign-in identity off the `users` row into its own table (one user, many
identities: provider, provider user id, login); sign-in resolves the user
through it. On Profile, a *Sign-in accounts* card lists GitHub, Google and
Discord as rows, only for providers the instance has turned on: a linked row
shows the account and an unlink button, an unlinked row a "Link <provider>"
button that runs that provider's OAuth in link mode and attaches the identity
to the signed-in user (no admission step). Unlinking the last identity is
refused and its row reads "Your only sign-in". An identity already attached
to another user is refused with a clear message.

The Profile card itself (display name, picture URL, Save in the footer)
already exists from ticket 16; keep it.

## Acceptance criteria

- [ ] Linking Google to a GitHub-created user lets that person sign in with either and land on the same user
- [ ] Unlinking works for any identity but the last; the last is refused server-side too
- [ ] Linking an identity that belongs to someone else fails without changing either user
- [ ] Providers the instance has not turned on are not listed

## Surfaces

- HTTP (list identities, start link, unlink); events for identity linked and unlinked
- Reverse state: unlink is the way back out of link
- Docs: `CONTEXT.md` (Sign-in identity), ADR 0040 amended for multiple identities

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/react-guide.md`,
`practices/testing.md`, ADR 0040, ADR 0061, ticket 03.

## Verification

`go test ./...`, `make lint`, `make coverage`, `make sqlc-check`; in `web/`: `bun run lint`, `bun run typecheck`, `bun run test`.

## Files likely touched

- `internal/auth/` (service, handler, model, the three provider clients)
- `internal/platform/storage/migrations/`, `internal/platform/storage/queries/auth.sql`
- `web/src/components/you/`, `web/src/hooks/AuthHooks.tsx`

**Size:** L
