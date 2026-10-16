# 14: Build container outages and their events

Type: task
Status: resolved
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

## Built

- Migration 0118 `service_outages`, one open outage per service by a partial unique index.
- A healthy deploy starts the 60-second wait for open outages of the services it brought up,
  since container events are held back during the deploy.
- The snapshot opens an outage only for an exit newer than the service's last closed outage, so
  a reconnect never re-raises an old one; compose one-off containers are ignored.
- Known limits, for the walkthrough (ticket 20) to judge: a person's `docker stop` (exit 143 or
  137) reads as a crash, and a late `die` from an old container just after a deploy can raise a
  false stop.
