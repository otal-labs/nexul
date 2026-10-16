# 15: Build watching a stack

Type: task
Status: resolved
Blocked by: 14

## What to build

From ticket 07: everyone who reads a stack watches it; a `stack_unwatchers` row in a new migration
marks an opt-out; a Watching toggle on the stack page; one notification when a service crashes
and one when it comes back, and the same for ticket 18's error-rate episodes once they land.

## Surfaces

- HTTP: `PUT` and `DELETE` on the stack's `watchers/me`, mirroring docs.
- MCP: `stack_update` takes `watching`; agents read the notifications with `notification_list`.
- UI: the toggle, placed where ticket 10 puts it; reverse state is turning it back on.
- Docs: the stacks and notifications guides.

## Built

- Migration 0119 `stack_unwatchers`; `watching` rides the stack's GET, `stack_get` and
  `stack_update` (which needs only read access when it changes `watching` alone).
- Notices go to the stack's workspace members minus opt-outs, read access checked again at write;
  `notifyStackWatchers` is the send path ticket 18 reuses. A stack outside every project has no
  watchers.
- No live topic: watching is the viewer's own state. The phone shows the notices and opens the
  stack, with no toggle of its own.
