# 30 — Deploys and runners on the phone

**Type:** implementation
**Status:** done
**Blocked by:** 25
**Decided in:** tickets 11, 12

## What to build

The Deploys tab: stacks with a status dot, name, target and last deploy
time; a stack screen with deploy history, services and containers read-only,
and Redeploy behind a confirm sheet; a deploy screen streaming the live log in
mono (virtualised, follows the tail). Under More: Runners with status dot,
name, machine and version. Same endpoints as the web stack, deploy and
runner hooks; live updates over the events client.

## Acceptance criteria

- [ ] Redeploy from the phone starts a deploy the web shows
- [ ] A running deploy's log streams live and follows the tail
- [ ] Runner status matches the web

## Surfaces

- UI (native)

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, `practices/borrowed-practices.md`, ticket 11.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/app/(tabs)/deploys/`, `native/src/app/(tabs)/more/runners/`
- `native/src/api/deploys.ts`, `runners.ts`

**Size:** L
