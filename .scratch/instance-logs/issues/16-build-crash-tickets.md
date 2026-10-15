# 16: Build the crash ticket switch

Type: task
Status: open
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
