# 01 findings: how the two new moments become events

## A ticket becomes unblocked

### Where "blocked" lives today

Nothing stores it. It is derived on every read from the `blocked_by` rows in
`ticket_links` plus the blocker's column stage.

- Board icon: `TicketBlockedLine.tsx:12` → `GET /api/tickets/blockers`
  (`internal/tickets/handler.go:108`) → `UnclearedBlockers`
  (`internal/tickets/ticket_links.go:258`) → `ListUnclearedTicketBlockers`
  (`queries/ticket_links.sql`, `COALESCE(s.kind, '') != 'done'`).
- Warning before a play: `web/src/hooks/useConfirmBlockedRun.tsx:11` reads
  `LinkSet.blocked` from `buildLinkSet` (`ticket_links.go:52-63`), where `Done`
  is `isDoneStage` (`internal/platform/storage/ticket_links_repo.go:127`).
- The run prompt (`internal/plays/run_links.go:59`) and MCP `waiting_on`
  (`internal/mcp/composite/tickets.go:173`) use the same reads. The web
  refreshes on `ticket.status_changed` and `ticket.link_*`
  (`web/src/hooks/useLiveEvents.tsx:136-139`).

### Every way a ticket can go from blocked to not blocked

1. A blocker enters a done-stage column: `Service.transition`
   (`internal/tickets/usecase.go:725`) → `TicketsRepo.UpdateStatus`
   (`internal/platform/storage/tickets_repo.go:157`), one transaction with the
   `ticket.status_changed` outbox row. Any move is allowed
   (`CanTransition`, `internal/tickets/model.go:60`); only a failed test refuses
   a done ticket (`internal/tickets/testing.go:195`). So a blocker can be dragged
   out of done and back in.
2. The last open blocker's link is removed: `RemoveBlocker`
   (`internal/tickets/ticket_links.go:242`) → `DeleteLink`
   (`ticket_links_repo.go:87`), which enqueues only when a row went away.
3. An open blocker ticket is deleted: `TicketsRepo.Delete`
   (`tickets_repo.go:231`). The link goes by `ON DELETE CASCADE`
   (`migrations/0010_ticket_links.sql:4`) and no `ticket.link_deleted` is
   published, so no consumer today can see this unblock.
4. A column's stage is redefined, for example progress to done:
   `RenameStatus` (`internal/workspace/usecase.go:927`, `updated.Kind = kind`).
   Every ticket in that column becomes done at once, in the workspace domain.

There is no bulk move. Deleting a column is refused while tickets use it
(`internal/workspace/usecase.go:982`), and a project with tickets cannot be
deleted. So paths 1–3 each change exactly one blocker per transaction. Path 4
is the only one that changes several.

### Recommendation

**Topic** `ticket.unblocked`, declared in `internal/tickets/events.go` `Topics()`.

**Payload** (thin, like `ticket.deleted`; matching reloads the ticket anyway
because the queue checks it again before it starts):

```json
{ "ticket_id": "…", "project_id": "…", "blocker_id": "…",
  "cause": "blocker_done | link_deleted | blocker_deleted",
  "actor": { "kind": "user", "user_id": "…" } }
```

`actor` reuses `tickets.Actor` (`model.go:111`). It holds the mover, the person
who removed the link, or the person who deleted the blocker, which is what
"whoever caused the moment" needs. When an automation caused it (a merged PR
moved the blocker), the actor is the automation and the matcher falls back to
the developer, as the decisions check already does
(`internal/plays/decisions_check.go:212`).

**Where it is published.** In the storage transaction that makes the change,
not in a consumer. One query, run inside that transaction *before* the
mutation, finds the tickets that wait on blocker X and on nothing else still
open:

```sql
-- name: ListTicketsWaitingOnlyOn :many
SELECT l.ticket_id, t.project_id FROM ticket_links l
JOIN tickets t ON t.id = l.ticket_id
LEFT JOIN statuses ts ON ts.id = t.status
WHERE l.kind = 'blocked_by' AND l.target_id = sqlc.arg(blocker_id)
  AND COALESCE(ts.kind, '') != 'done'
  AND NOT EXISTS (
    SELECT 1 FROM ticket_links o JOIN tickets b ON b.id = o.target_id
    LEFT JOIN statuses s ON s.id = b.status
    WHERE o.ticket_id = l.ticket_id AND o.kind = 'blocked_by'
      AND o.target_id != sqlc.arg(blocker_id) AND COALESCE(s.kind, '') != 'done');
```

It runs only when X is open now and the change stops X counting:

- `UpdateStatus`: the old stage is not done and the new one is. Both are read
  in the transaction; `GetTicketStatusAndCategory` already reads the old status.
- `DeleteLink` for a `blocked_by` link: X is not done. Keep only the linked
  ticket from the result.
- `Delete` of X: X is not done. Run the query before the `DELETE`, because the
  cascade removes the rows it reads.

The domain owns the payload. The repo method takes a builder,
`func([]UnblockedTicket) []eventbus.OutboxEvent`, and calls it inside the
transaction, the same idea as `transition`'s `extra ...func(Ticket)`
(`usecase.go:725`). This needs no new seam: the outbox, its relay, and the
consumer dedupe (`practices/architecture.md` §3, §6) do the rest.

**Why once per unblock.** The event and the change commit together, so a
rollback loses both and a commit keeps both. A repeated HTTP move is a no-op
(`usecase.go:733` returns early on the same status). Removing a missing link
enqueues nothing (`ticket_links_repo.go:87-108`). A redelivered outbox row is
deduped per consumer under its row ID (ADR 0018). SQLite's single writer
serialises two people finishing the last two blockers at the same moment. The
first move still sees the other blocker open, and only the second fires.
Checking in a consumer afterwards (the decisions-check style) was rejected. It
races with exactly that pair, and it cannot see path 3 at all.

**Edge cases**

- Blocker moved out of done, then back: each entry into done fires. Leaving
  done publishes nothing (no `ticket.blocked` topic yet). The second fire is a
  real second unblock; "once per ticket per occurrence over a period" belongs
  to the auto play's limits (tickets 03 and 10), not to the event.
- Done to done, or removing a link to an already-done blocker: nothing fires.
- Blocked ticket deleted: its links cascade and nothing fires.
- Blocked ticket itself in done: filtered out (`ts.kind`). Done tickets are
  never reopened (ADR 0064), so "can start now" means nothing for them.
- Several blockers in one transaction: impossible on paths 1–3 today. If a bulk
  move is added, collect the ids into a set and fire once per ticket.
- Column stage redefined (path 4): do not fire. It is board configuration in
  another domain, not a ticket reaching done. Say so in the catalog
  description.

## A doc is created or changed

### Topics and attribution today

- Doc topics: `doc.created`, `doc.updated`, `doc.deleted`, `doc.moved`, the
  folder and watcher topics, and eight clarification topics
  (`internal/docs/events.go:6-31`). `doc.updated` also fires for archive and
  lock (`internal/docs/usecase.go:338`, `:388`, `lock_changed`).
- Title and body writes: `Update` (`usecase.go:271`, from HTTP and MCP
  `doc_update`), `CommitCollab` (`usecase.go:509`, the live editor, one
  `doc.updated` per converged commit, `internal/collab/session.go:181`),
  `WriteNoGaps` (`internal/docs/clarification.go:324`, MCP only). Creates go
  through `CreateInFolder` and `Clone` → `DocsRepo.Create`
  (`internal/platform/storage/docs_repo.go:24`).
- Attribution is only `actor_id`, and it is the person: ADR 0101 says an edit
  made over MCP or by a play acting for someone is attributed to that person.
  `identity.Actor` (`internal/platform/identity/identity.go:7`) has an ID and an
  optional automation, and nothing else.
- How agent writes reach the server: `/mcp` is `RequireAuth(withIdentity(...))`
  (`server/cmd/routes.go:227`) with the computer's personal access token
  (`server/cmd/wire_pairing_tokens.go:17`). The only agent doc writes are
  `internal/docs/mcp.go:198`, `:261`, `:297`. A play run's agent and a person's
  own agent session use the same token, and no request names a trail, so the
  two cannot be told apart. The live editor needs a signed-in WebSocket
  (`internal/collab/hub.go:69`), so a collab commit is always a person.
- A precedent for marking MCP writes already exists:
  `tickets.CreateOptions.ViaMCP` (`internal/tickets/usecase.go:140-147`), set by
  `internal/mcp/composite/tickets.go:404`, and `testActor`'s `viaMCP`
  (`internal/tickets/testing.go:228`).

### Timers and seams

There is no generic scheduler or durable debounce. In-memory timers
(`internal/plays/run.go:950`, `internal/presence/keeper.go:171`) die with the
process. The durable pattern is a due column polled by a ticker: the webhook
relay's `Due(now)` (`internal/integrations/webhook.go:193-213`), with tickers
started in `server/cmd/workers.go:83-84`. Exactly-once is a conditional write
that enqueues only when it took effect (`SetFinishedAt`,
`internal/tickets/usecase.go:946`).

### Recommendation

**Agent rule.** An edit or create counts only when a person made it: an actor
is present, it is not an automation, and it did not come over MCP. The MCP
adapter passes `ViaMCP` the way the tickets adapter does. This covers a play
run's agent and any other agent alike, which matches CONTEXT.md's "not by an
agent". An automation-token write does not count either: a doc has no
developer to fall back to, so it would always be "didn't run".

**Durable settle, no in-memory timer.** Add one table in a new forward-only migration:

```sql
CREATE TABLE doc_settles (
    doc_id   TEXT PRIMARY KEY REFERENCES docs(id) ON DELETE CASCADE,
    due_at   INTEGER NOT NULL,
    actor_id TEXT NOT NULL,
    first    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_doc_settles_due ON doc_settles(due_at);
```

- `DocsRepo.Create` inserts `due_at = now + 10m, first = 1`. `Update` and
  `CommitBody` upsert `due_at` and `actor_id` and leave `first` unchanged on
  conflict (`ON CONFLICT(doc_id) DO UPDATE`). These writes go in the
  transactions that already add the editor as a watcher (ADR 0101), so they
  cost one row per save. The use case passes the settling person's id, or an
  empty one for MCP and automation writes. Archive, lock, move, and a save that
  changes neither title nor body skip it.
- `docs.RunSettleLoop(ctx, svc, 30*time.Second)` is started in
  `server/cmd/workers.go` beside the cleanup loops. Each tick, in one
  transaction, it selects due rows joined to non-archived docs, deletes every
  due row, and writes one `doc.settled` outbox row per kept doc.

**Topic** `doc.settled`, declared in `internal/docs/events.go`:

```json
{ "doc": { "id": "…", "project_id": "…", "title": "…" }, "first": true, "actor_id": "…" }
```

`doc` reuses `WatchedDoc`. The moment "doc is created" matches `first: true`,
and "doc is changed" matches `first: false`. `actor_id` is the last person who
edited, which "whoever caused the moment" needs. `doc.created` and
`doc.updated` stay as they are for notifications and automations.

**Restart.** The rows are in SQLite. After a restart the first tick fires
everything that came due while the server was down, at most one tick late.
A crash mid-tick rolls back both the delete and the outbox rows, so nothing is
lost or doubled.

**Exactly once.** Deleting the row and inserting the outbox row commit together, and the single
writer serialises a tick against a save. A save that lands after the tick
inserts a fresh row with `first = 0`, which fires later as a change. One server
owns the database (ADR 0011), so two pollers never compete.

**Created and then edited within 10 minutes** fires once, as created
(`first: true`), 10 minutes after the last edit. A created moment that fired at
once would run a play on a title and an empty body, because the web creates the
doc first and the person then types into it. Edits after that settle fire
`first: false`.

**Other cases.** An agent edit inside a person's window leaves the row alone,
so the person's settle fires on time. A doc an agent created and a person
later edited settles once with `first: false`. Deleted while pending: the
cascade drops the row. Archived while pending: the tick drops it with no event.
Moved while pending: the payload is built at fire time, so it names the current
project.

## What each new topic must touch

Each topic is declared in its domain's `Topics()` with its payload type, and
`make event-schemas` regenerates the published schema and the SDK types
(ADR 0044, ADR 0137). It needs a scope rule in `server/cmd/automation_scope.go`:
`projectScope` for `ticket.unblocked`, and `nestedProjectScope("doc")` for
`doc.settled`. CONTEXT.md needs a line under "Blocked by" and "Auto play". A
browser live push is optional: `ticket.status_changed` already refreshes the
blocked icon on path 1, and nothing on screen shows the settle.

## Open risks

- Blocker deletion still publishes no `ticket.link_deleted` for the cascaded
  links, and `ticket.deleted` is not in `useLiveEvents.tsx`, so an open board
  keeps the blocked icon until a refetch. This is an existing gap and out of
  scope here; ticket 08 can live-push `ticket.unblocked` to the blockers key
  and close it.
- A doc edited at least once every 10 minutes never settles until the edits
  stop. That is what "edits have stopped" means, but a long writing session
  shows nothing for hours.
- A person's own agent editing over MCP never counts, even when the person
  thinks of it as their own edit. The rule is simple and has no exceptions;
  confirm it in the walkthrough.
- The "Clarify via AI on doc changed" loop: the run locks the doc (ADR 0121),
  so the person cannot extend the window while it runs. A person edit just
  before the run still settles afterwards. The per-ticket-style concurrency cap
  needs a per-doc twin in ticket 03.
- A column's stage being redefined unblocks tickets silently (path 4). This is
  rare, and it is documented rather than handled.
- A thin `ticket.unblocked` payload is permanent once published (ADR 0044).
  Fields can be added later, never renamed.
