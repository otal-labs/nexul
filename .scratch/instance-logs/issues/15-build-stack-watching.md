# 15: Build watching a stack

Type: task
Status: open
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
