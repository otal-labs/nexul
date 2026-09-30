# 07 — Container logs on the phone

**Type:** task
**Status:** resolved
**Blocked by:** 01, 04, 06

## Question

What the phone app shows: a tap on a service row in the stack screen opening a read-only log view that follows the tail, the snapshot only, or nothing for now. Decided on the owner's delegation for the phone app.

## Answer

Decided 2026-09-30 on the owner's delegation for the phone app.

- Tapping a service row on the stack screen opens a read-only log screen.
  It follows the tail over the same WebSocket as the web, using the phone's
  session token.
- Scrolling up pauses following, and returning to the bottom resumes it.
  There is no search or stream filter; stderr is marked the same way as on
  the web.
- It is gated by `stacks:logs`, and the row shows no affordance without it.
