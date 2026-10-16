# 16: Build the crash ticket switch

Type: task
Status: resolved
Blocked by: 14

## What to build

From ticket 07: the stack setting "Open a ticket when a service crashes", off by default; one
ticket per crashed service in the stack's project (default type, first stage, nobody assigned,
created by the instance); a repeat crash or a "came back" adds a comment to the open ticket.
Ticket 18's episodes reuse it.

## Surfaces

- MCP and HTTP: the switch on `stack_update` and the stack's settings route.
- UI: the switch, placed where ticket 10 puts it.
- Docs: the stacks guide.

## Built

- Migration 0120: `stacks.crash_tickets` and a `stack_crash_tickets` link keyed by stack and a
  subject (the service today; ticket 18 picks its own subject) through `FileOrComment` and
  `CommentOpen`.
- Comments are system messages in a thread owned by the workspace's first active Owner, since a
  thread needs a real user. A bug default type files with its origin marked unknown.
- A ticket in a done stage means the next crash files a new one; a recovery leaves it open.
- Left open: a failed link write right after a successful create makes the retry file a second
  ticket; closing it needs a tickets API taking a caller-made id.
