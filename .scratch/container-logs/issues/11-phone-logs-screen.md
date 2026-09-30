# 11 — Logs on the phone

**Type:** task
**Status:** ready-for-agent
**Blocked by:** 08

## What to build

Per decision 07:

- Tapping a service row on the phone's stack screen opens a read-only log
  screen. It follows the tail over the same WebSocket, using `?token=` with
  the session token.
- Scrolling up pauses following, and a "Jump to live" pill resumes it.
- stderr lines are marked the same way as on the web.
- The row has no affordance without `stacks:logs`.
- It uses Legend List.

## Acceptance criteria

- [ ] Component tests with a fake socket: render, pause, and close on blur.
- [ ] Checked on the emulator through the device panel.
- [ ] `bun run lint`, `typecheck` and `test` are green in `native/`.
