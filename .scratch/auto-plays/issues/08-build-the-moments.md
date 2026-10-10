# 08: Build the unblocked and doc-changed events

Type: task
Status: resolved
Blocked by: 01

## Question

Build what 01 found: the `ticket.unblocked` topic and a settled doc-change
topic, each with a catalog row, an outbox write, and Go tests, plus the
SDK's generated topics (`sdk/src/events.generated.ts`) so automations can
listen to them too.

## Answer

Built in PR #522. `ticket.unblocked` (payload `ticket_id`, `project_id`,
`blocker_id`, `cause` of `blocker_done` / `link_deleted` /
`blocker_deleted`, `actor`) is written in the transaction of the change on
all three paths. `doc.settled` (`doc`, `first`, `actor_id`) comes from the
`doc_settles` table (migration 0081) and `docs.RunSettleLoop`, which sleeps
until the earliest window closes or a commit lands. MCP, automation and
server-side writes never open a window; a save that changes neither title
nor body doesn't either.
