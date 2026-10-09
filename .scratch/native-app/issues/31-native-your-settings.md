# 31 — Your settings on the phone

**Type:** implementation
**Status:** done
**Blocked by:** 25
**Decided in:** tickets 11, 12

## What to build

Under More: Your settings with Profile (avatar, display name, sign-in
accounts, read-only), Appearance (system, light, dark, persisted), Devices
(the same list as the web: current device first, others with sign out, Sign
out everywhere else), a workspace switcher sheet that scopes every tab, and
Sign out (deletes the current session server-side, clears secure store, back
to first-run). The More tab lists Docs, Runners, Your settings and shows the
instance host and app version at the bottom.

## Acceptance criteria

- [ ] Signing this phone out from the web returns the phone to first-run on its next call
- [ ] Switching workspace changes what every tab shows
- [ ] Appearance persists across launches

## Surfaces

- UI (native)

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, tickets 05 and 11.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/app/(tabs)/more/`
- `native/src/api/sessions.ts`, `native/src/stores/`

**Size:** M
