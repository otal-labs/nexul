# 14: Build container outages and their events

Type: task
Status: open
Blocked by: None — can start immediately

## What to build

Ticket 05, as decided: machine runners forward Docker's die, oom and start events and a reconnect
snapshot; the server matches them to stacks, ignores deploys and deletes, stores
`service_outages` in a new forward-only migration, closes one after 60s running, and publishes
`service.container_stopped` and `service.container_recovered` with no output lines.

## Surfaces

- Events: `Topics()`, outbox, `make event-schemas`, live push with a `stacks:read` audience,
  `make live-topics`, the web follower and the phone query's `refreshes`.
- MCP: `stack_get` returns open outages.
- UI: the services list shows an open outage, placed where ticket 10 puts it.
- Automations: the generated SDK topics.
- Docs: the stacks guide and the automations events list.
