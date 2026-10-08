# 08: Build the unblocked and doc-changed events

Type: task
Status: open
Blocked by: 01

## Question

Build what 01 found: the `ticket.unblocked` topic and a settled doc-change
topic, each with a catalog row, an outbox write, and Go tests, plus the
SDK's generated topics (`sdk/src/events.generated.ts`) so automations can
listen to them too.
