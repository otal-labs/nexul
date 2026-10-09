# 28 — Board and ticket on the phone

**Type:** implementation
**Status:** done
**Blocked by:** 25
**Decided in:** tickets 11, 12

## What to build

The Board tab: a project picker sheet, then tickets as sections per status
(the board's column order) with a "Mine" filter; rows show mono ticket id,
title, type and label dots. The ticket screen: id, title, status, type,
assignees, the markdown body, a status sheet that changes it, "Assign to me",
and "Open thread" into Chat. Same endpoints as the web board and ticket
hooks. Live updates over the events client.

## Acceptance criteria

- [ ] Changing status on the phone moves the ticket on the web board live
- [ ] Mine shows only the viewer's tickets
- [ ] Sections follow the project's status order

## Surfaces

- UI (native)

## Read first

`AGENTS.md`, `practices/native.md` (written by ticket 24), `practices/react-guide.md` (F1–F7 and the self-review checklist, applied to React Native), `practices/testing.md`, ticket 11.

## Verification

In `native/`: `bun run lint`, `bun run typecheck`, `bun run test`. Screenshots from an Android emulator or a web preview of the screens at a phone width, kept out of git.

## Files likely touched

- `native/src/app/(tabs)/board/`
- `native/src/api/tickets.ts`

**Size:** L
