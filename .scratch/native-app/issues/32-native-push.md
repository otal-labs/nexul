# 32 — Push notifications on the phone

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 23, 25
**Decided in:** ticket 13

## What to build

After sign-in, ask for notification permission once (Android 13+), get the
Expo push token with the project id from app config, and register it with
`PUT /api/auth/sessions/current/push-token`; clear it on sign out. Tapping a
notification opens the Inbox item named by `notification_id` (cold start and
warm). A missing project id (owner setup not done) skips registration and
logs once, never crashes.

## Acceptance criteria

- [ ] With a project id configured, a notification created on the server reaches the emulator and opens the right item
- [ ] Without one, the app runs normally

## Surfaces

- UI (native)
- Uses HTTP from ticket 23

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, `practices/borrowed-practices.md`, ticket 13.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/push/`
- `native/app.config.ts`

**Size:** M
