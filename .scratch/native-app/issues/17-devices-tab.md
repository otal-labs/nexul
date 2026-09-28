# 17 — Devices tab

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 15, 16
**Decided in:** tickets 02, 05

## What to build

The Devices tab in Your settings → Security, in front of Tokens, exactly as
locked in ticket 05: the *Connect the desktop app* card (one click mints and
copies the connection token, the icon swaps to a green check with "Copied"
for two seconds, a failure shows an error toast), and the *Signed-in
devices* card listing sessions from ticket 15: the current device boxed
alone with a muted mono "THIS DEVICE" tag, other devices as hairline rows
with a ✕ that arms a "Sign out" confirm, and a footer with "Sign out
everywhere else". An EmptyRow when no other device is signed in. The list
updates live on `session.created` and `session.revoked`.

Leave a slot for the *Connect a phone* card, which ticket 18 fills; until
then the desktop card sits alone at full width.

Motion per ticket 05's micro list (row exit, sign out everywhere else, copy
tick); the connected hero belongs to ticket 18.

## Acceptance criteria

- [ ] The list shows every session of the signed-in user with label, IP and relative last active, the current one first
- [ ] Signing a row out removes it with the exit motion and the device is signed out server-side
- [ ] A device signing in elsewhere appears without a refresh
- [ ] Copy connection token puts a working token on the clipboard and shows the tick
- [ ] Reduced motion leaves everything in its final state

## Surfaces

- UI; live WebSocket updates; HTTP from ticket 15
- Reverse state: sign out is the way out of a session; signing in again is the way back

## Read first

`practices/react-guide.md`, `practices/design-language.md`, `practices/testing.md`,
ticket 05's answer, and branch `proto/your-settings` for the built reference.

## Verification

In `web/`: `bun run lint`, `bun run typecheck`, `bun run test`; screenshots at 768, 1024, 1440px in both themes.

## Files likely touched

- `web/src/components/you/` (or the settings folder the split settled on)
- `web/src/hooks/AuthHooks.tsx` or a new `SessionHooks.tsx`, `web/src/models/Session.tsx`
- `web/src/api/ws.tsx` (event fan-out)

**Size:** M
