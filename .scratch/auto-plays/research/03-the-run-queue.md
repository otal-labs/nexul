# 03: The run queue — findings

Answers ticket 03. Paths are relative to the repo root.

## How things work today

### How a play run starts

- A press goes through `Runner.Run` (`internal/plays/run.go:321`): `checkPlayAndTarget`
  (`run.go:734`) reads the play and target, then `checkPlay` (`run.go:761`) checks the
  workspace, enabled, project membership, excluded projects, `plays:run` on the play (this is
  where a permission overwrite excludes a person, `run.go:778`), and the ticket play's
  show-when stage (`run.go:781`).
- `launch` (`run.go:345`) does the rest, in this order: `HarnessResolver.ResolveTarget`
  (`run.go:353`), then `refuseIfActive` (`run.go:363`, one active trail per target), memories,
  thread, `CreateTrail`, the "Started" message, then `startTurn`, which runs the turn in a
  detached goroutine (`run.go:941-961`). `launch` returns as soon as the turn has started; it
  does not wait for the turn to finish.
- A harness refusal always becomes a failed trail (`run.go:354-360`). Other refusals do too
  when `recordRefusals` is true, which only the decisions check sets.
- The decisions check is the only automatic start today. `HandleTicketStatusChanged`
  (`internal/plays/events.go`) is subscribed as consumer `plays.decisions_check`
  (`server/cmd/subscriptions.go:383`). From there `onTicketMoved`
  (`internal/plays/decisions_check.go:148`) runs `enteredDone`, then `hasDecisionsCheck` (once
  per ticket, ever, `:198`), then `decisionsStarter` (the mover, else the developer, `:212`),
  then `startDecisionsCheck(record=true)` (`:85`). It launches inline, inside the event handler,
  and logs and swallows any start error (`:164-166`), so the bus never retries a check that
  failed to start.
- "Decisions check didn't run" is just the latest decisions-check trail in state `failed`
  (`web/src/hooks/TrailHooks.tsx:109-115`, `web/src/components/play/DecisionsCheckNotice.tsx`).
  Its retry button calls `POST /api/plays/decisions-check` (`internal/plays/run_handler.go:45`),
  which runs `RetryDecisionsCheck` (`decisions_check.go:76`) on the viewer's own harness without
  recording refusals. MCP reaches the same retry through `play_run` with `decisions_check: true`
  (`internal/plays/run_mcp.go:83`).

### How the server knows a run ended

- Every terminal path goes through `finish` (`run.go:1257`), which saves the trail together
  with a `play.run_finished` outbox event (`run.go:1266`, `:1321`). The paths are the pipeline's
  `OnFinished` (`:1121`), silence (`:1160`), Stop with or without a live observer
  (`:1178`, `:1225`), and a restart that cannot reattach (`:1243`).
- A refusal recorded with `createFailed` (`run.go:1270`) emits `play.run_finished` only when
  the trail has a starter (`:1274-1277`).
- `waiting` is active and emits no finish (`run.go:1134-1137`; `TrailState.Active` in
  `internal/plays/model.go:102`).
- After a restart, `ResumeRunsAfterRestart` (`run.go:1230`, ADR 0119) runs at boot before the
  decisions-check subscription is registered (`subscriptions.go:375` then `:383`). It follows
  `running` trails again and ends `starting` ones as interrupted. Once it returns, the set of
  active trails is accurate, so a dispatcher that starts after it can count active trails and
  trust the number.

### What "computer online" means to the server

- There is no stored or continuously tracked "online" state. The presence keeper holds harness
  connections only while the owner has a browser socket open (`internal/presence/keeper.go:1`,
  `:128`, `:154`). Its one hook, `SetOnlineChanged`, is about the browser, not the computer,
  and the live hub already uses it (`subscriptions.go:333`).
- A computer is reachable at any time through its computer tunnel (ADR 0062), whether or not a
  browser is open. The real readiness test happens at launch. `ResolveTargetOverride`, then
  `requireSetup` (`internal/pairing/usecase.go:533`, `:592`), calls the harness's
  `ListProviders`. If that call fails, the result is `NotConfiguredError{Reason: ReasonOffline}`
  (`usecase.go:602-603`, `internal/pairing/model.go:340`). `playsHarnessResolver`
  (`server/cmd/wire_plays.go:76-86`) passes the reason on as `HarnessRefusal.Reason`.
- Nothing announces a computer coming back online. The tunnel watch polls Cloudflare only while
  a computer is being paired (`internal/pairing/tunnel_watch.go:41`, `server/cmd/workers.go:31`).
  So "online" can only be learned by trying, and the only way to notice a computer coming back
  is to try again on a timer.

### Facts that shape the design

- Only one run at a time per target already holds for every run, manual or automatic
  (`refuseIfActive`, `run.go:809-820`). That makes the per-ticket slot free: it is 1, and it
  covers manual runs too.
- If a person never set a start-in, their runs start in the project folder
  (`startsInWorktree` returns true only for an explicit `worktree`, `pairing/usecase.go:677-693`;
  CONTEXT.md "Start-in"). Two runs at once for that person edit the same checkout.
- A run has no time limit; only 15 minutes of harness silence ends one (`run.go:32`).
  A `waiting` run pauses the silence clock (`run.go:1063-1069`), so it can stay active for hours.
- A run's own edits are not marked. The agent's MCP calls come in under the person's computer
  token (`internal/pairing/mcp_token.go:11`), so a card the agent moves arrives with actor kind
  `user:mcp` and the starter's user id. `ActorKindPlay`, which has a `TrailID`, exists
  (`internal/tickets/model.go:111-127`), but nothing sets it, because a play no longer moves
  its own ticket (ADR 0112).
- A ticket play only runs in its show-when stage (`run.go:781`). "Fix with AI" is a
  progress-stage play, so firing it when a ticket in backlog becomes unblocked would be refused.
  See the risks.

## Recommendation

### The queue table (plays domain, new migration, sqlc queries in `play_queue.sql`)

`play_queue`: one row for each time a moment matches, kept after the run is decided, so the same
table drives the ticket's signals, the daily cap, and per-auto-play limits.

| column | meaning |
|---|---|
| `id` | row id |
| `workspace_id`, `project_id`, `target_type`, `target_id` | where the run goes |
| `play_id`, `origin_kind`, `origin_id` | the play, and what queued it: `auto_play` plus its id, or `automation` plus its id (for `runPlay`, ticket 14) |
| `person_id` | who it runs on, resolved when the moment matched |
| `run_on` | `developer`, `tester`, `causer`, used again in the re-check |
| `moment`, `cause` (JSON) | the moment and its actor and via, for the re-check and for `Via` on the trail |
| `priority` | 2 High, 1 Normal, 0 Low |
| `status` | `queued`, `dispatching`, `started`, `skipped`, `didnt_run`, `cancelled` |
| `reason` | why it is waiting (`offline`, `ticket busy`, `paused`) or why it was skipped or didn't run |
| `trail_id` | set from `dispatching` on |
| `queued_at`, `decided_at`, `not_before` | times; `not_before` holds back an item whose computer was offline |

Indexes: `(person_id, priority DESC, queued_at) WHERE status='queued'` for the dispatcher;
`(target_type, target_id, decided_at)` for the ticket feed and the cap; and a unique
`(origin_id, target_id) WHERE status IN ('queued','dispatching')`. The unique index makes a
second match while one is already waiting a no-op. That bounds the queue to one item per auto
play per ticket, and it makes the matching consumer idempotent the way architecture section 6
asks.

Also `play_queue_resumes(target_type, target_id PRIMARY KEY, resumed_by, resumed_at)` for the
resume button (see the daily cap).

The bus cannot hold this queue: it has no durable `Enqueue`/`Consume` (see the risks), and
priority order with per-person slots needs its own table anyway.

### The dispatcher loop and what wakes it

One goroutine, `Runner.RunQueue(ctx)`, started in `startBackgroundWorkers` after the
subscriptions are registered and after `ResumeRunsAfterRestart`. It needs a shutdown path through
`ctx` (go.md section 7). It waits on a `kick chan struct{}` with a buffer of 1 (sends never block)
and on a one-minute ticker.

Things that kick it:
- A row is queued: the matching consumer kicks after it inserts.
- `play.run_finished`: a new subscription, consumer `plays.queue`. Every way a run ends already
  publishes it.
- Resume pressed, or a queued run cancelled: the use-case kicks in-process.
- Boot: one pass at start. Before that pass, a row left in `dispatching` becomes `started` if
  its `trail_id` exists, and goes back to `queued` if not. Because the trail id is written onto
  the row before `launch`, a crash anywhere during the start leaves no duplicate run.
- The ticker. This covers a computer coming back online and `not_before` passing. No new
  presence hook is needed. Waking on the keeper's connected state would cut the wait from up to
  a minute to seconds. Add it if the minute ever matters, by composing with the existing
  `SetOnlineChanged` hook.

One pass:
1. Read the people who have `queued` rows with `not_before <= now`.
2. For each person, count free slots: the cap minus their active trails. That needs a new query,
   `CountActivePlayTrailsByStarter`, served by the existing `idx_play_trails_choices` index on
   `starter_id`. Every trail counts, manual or automatic. Chat `@Agent` turns are not trails and
   do not count.
3. Go through their queued rows, highest priority first and oldest first within a priority,
   until the free slots run out:
   - If the ticket already has an active trail, set `reason='ticket busy'` and skip to the next
     row. A busy ticket must not hold up the person's other tickets.
   - If the ticket is paused, set `reason='paused'` and skip to the next row.
   - Run the re-check below. If it fails: `skipped`, with the reason.
   - Otherwise write `dispatching` with a new `trail_id`, then call `startQueued`, which works
     like `startDecisionsCheck` generalized: `identity.WithActor(starter)`, `checkPlay`, a trail
     with the preset id and the cause's `Via`, then `launch(record=true)`.
   - If the harness refusal reason is `offline`: back to `queued`,
     `reason='offline'`, `not_before = now+1m`, and stop this person's pass. That is a
     simplification with a ceiling: a person whose project links point at two computers waits
     on the offline one too.
   - Any other refusal: `didnt_run`, with the failed trail `launch` already wrote.
   - Started: `started`.
4. Bound each attempt with `context.WithTimeout` (30s) so a hung tunnel cannot stall the loop.

New seam: in `launch`, an `offline` refusal on a queued start must not write the failed trail it
writes today (`run.go:359`). Pass a flag, or return the refusal before `createFailed` when the
trail came from the queue. Everything else in `launch` is reused unchanged.

One goroutine makes every slot decision, so auto runs never race each other. A manual press can
race the dispatcher on one ticket; that read-then-create race already exists between two manual
presses (`refuseIfActive`).

### Default concurrency per person: 1

- The default start-in is the project folder, so a second run at the same time would edit the
  same working tree as the first.
- Runs are long and open-ended (no ceiling, a 15-minute silence clock, `waiting` can last hours),
  and every run spends the person's own provider quota.
- The per-target rule already makes each ticket sequential, so a chain (test fails, then
  Fix with AI, then review) runs in order anyway.

Ship it as a constant for the first cut. When someone who works in worktrees asks for more, it
becomes a number on the person's pairing defaults, next to start-in, because it describes their
computer, not the workspace. Ticket 02 decides where settings live; this would be the only
per-person one.

The per-ticket limit is 1 and not configurable, because it is the existing one-active-trail rule.

### Manual presses

Manual presses never queue and are never refused for lack of a slot. They do take a slot, because
the count in step 2 includes every active trail. Auto runs wait behind manual ones; never the
other way round. A manual press on a ticket that has a queued auto run starts normally. The
queued row then waits as `ticket busy` and re-checks afterwards.

### The re-check at the front

Check again, against current data:
- the play: `checkPlay` (enabled, project not excluded, the person still has `plays:run`, the
  stage gate);
- the auto play: still enabled, conditions still match, the moment still holds ("still
  unblocked", "still in stage X", "still has this developer");
- the person: the `run_on` role still names the same person. If not, skip ("no longer
  alice's ticket"). The new assignment raises its own moment.
- the auto play's own limit;
- duplicates: if a trail of the same play started on the target after `queued_at`, skip
  ("already ran"). This covers a manual press of the same play made while the item waited.

Each moment needs a "still holds" check; ticket 08 should define one next to each moment.
`play_id` deleted: the row is `skipped`.

### The daily cap and pause

- Count: rows with `status='started'` on the ticket and
  `decided_at > max(now-24h, resumed_at)`. The window rolls over 24 hours rather than following
  calendar days, so there are no time zones to handle. Only started runs count; skipped and
  didn't-run rows did no work.
- Paused is computed, never stored: paused means the count is at least the cap (default 5).
  Ticket 02 decides where the number is kept.
- While paused, matching still queues, at most one row per auto play per ticket because of the
  unique index. The dispatcher passes over those rows with `reason='paused'`, and the ticket
  shows them under the "Auto plays paused on this ticket" banner.
- Resume upserts `play_queue_resumes` (`autoplays:write` or the ticket's developer; ticket 02 to
  confirm), which resets the count to zero from now, and kicks the dispatcher. Resume clears
  nothing else: the queued rows run after their re-check.
- Runs started through `runPlay` count toward the cap; manual presses do not.

### Didn't run versus skipped

- **Didn't run**: a failed trail, written by the `createFailed` and `launch` paths that exist
  today, plus the row in status `didnt_run` with `trail_id` and `reason`. It covers:
  - nobody to run on (the trail has no starter, and no finish event, as today);
  - the person excluded from the play;
  - a harness refusal other than offline: unpaired, expired token, no default computer or
    project, setup required, or a protocol error.

  Those last cases go beyond the map's "only nobody, or excluded" list. Each one needs the
  person to fix something, so a run waiting on them silently would never start. They are also
  exactly the cases the current "Decisions check didn't run" notice exists for (ADR 0066, and
  `SetupRefusalLink` in the notice). An offline computer and a busy ticket only queue.
- **Skipped**: no trail, because nothing ran. The row has status `skipped` and a reason. The
  ticket shows a muted line: "Fix with AI skipped: no longer unblocked".
- The ticket reads one feed: the target's queue rows newest first, plus the paused flag and
  count, kept live by an ephemeral `play.queue` frame (the `TopicPlayRun` pattern,
  `events.go:24`). "<Play> didn't run" with "Run again" generalizes the current notice; the
  button presses the play as the viewer through the normal `Run`.

### The decisions check on this path

| today | on the queue |
|---|---|
| consumer on `ticket.status_changed`, launched inline in the handler | moment "enters stage done" queues a row; the handler only inserts |
| mover, else the developer when an automation moved it (`decisionsStarter`) | run on "whoever caused it", same rule |
| once per ticket, ever (`hasDecisionsCheck`) | the seeded auto play's limit "once per ticket"; done never reopens (ADR 0064) |
| workspace switch (`workspaces.decisions_check_enabled`) | the seeded auto play's enabled flag, carried over by ticket 13's migration |
| nobody, or no `plays:run`: failed trail | didn't run (same) |
| setup refused, unpaired: failed trail | didn't run (same) |
| computer offline: failed trail | **queues** until the computer answers |
| another run on the ticket: failed trail | **waits** for the ticket's slot |
| retry on the viewer's harness | "Run again" presses the now-ordinary play |

Offline and busy no longer fail the check; ticket 13 says so in the user guide.

### Loops

The daily cap is enough, so mark nothing new for loops.
- Chains across plays are wanted, and the per-ticket slot already runs them one after another.
- Five runs per ticket per day caps the worst loop at five runs of one person's quota, after
  which the ticket shows the paused banner.
- An auto play re-firing on its own run's change is cut short by its "once per occurrence"
  limit and by the "already ran" re-check.
- "Whoever caused it" already resolves to the starter for an agent's move, because `user:mcp`
  carries the starter's user id, so a chain stays on one person.

Ticket 01 needs a run's edits marked for "doc changed, not by an agent". If 01 adds that marker
(for example `ActorKindPlay` with `TrailID`), the queue can store it in `cause`, but nothing here
depends on it.

## Open risks

- **The stage gate versus moments.** `checkPlay` refuses a ticket play outside its show-when
  stage (`run.go:781`). "Run Fix with AI when the ticket becomes unblocked" would be skipped
  for every backlog ticket. Either the composer only offers moments and conditions that imply
  the play's stage, or auto runs skip the stage gate. That is a product decision for 02, 05, or
  07; the queue only reports the skip.
- **Waiting runs hold the slot.** A run parked on an unanswered question keeps the person's
  single slot, which stalls their queue until they answer or stop it. The ticket feed should say
  which run holds the slot. Not counting `waiting` would bring the shared-folder problem back.
- **Offline detection costs a harness call**, at most one `ListProviders` per waiting person
  per minute, since the pass stops at the first offline result.
- **History grows.** Rows are kept so "once per ticket" limits work forever; prune `skipped`
  and `cancelled` rows if the table ever matters.
- **Who may cancel a queued run or resume a ticket** is not settled here. The proposal is the
  person it runs on, or an `autoplays:write` holder; ticket 02 to confirm.
- **Docs drift.** `practices/architecture.md` section 2 describes `Enqueue` and `Consume` on the
  bus, which do not exist. Fix that with this effort's docs pass.
