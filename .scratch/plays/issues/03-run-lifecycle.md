# 03 — The run lifecycle and record

**Type:** grilling
**Status:** resolved
**Blocked by:** 02

## Question

A run is one press of a play button. Settle its life from click to record.

1. States: `queued`? `running`, `done`, `failed`, `interrupted`. Does a run
   exist before the harness accepted the turn, so a "harness refused to
   start" failure is recorded too?
2. Record fields: play, target (ticket or doc), starter, selected memories,
   custom instructions, the move-to column chosen, harness session id,
   started, ended, outcome, and last error. The record is also what the
   dialog reads to pre-select the user's last choices. Anything the owner needs for
   "use Nexul to fix Nexul" debugging that this misses?
3. Concurrency: one active run per target, button disabled while running.
   Can a second user start a different play on the same ticket? Who may
   interrupt: the starter only, or anyone with `plays:write`?
4. Success and move-to: the server moves the ticket to the column chosen in
   the dialog when the harness reports the turn done. If the chosen column
   was deleted mid-run, skip the move and say so in the thread. If the Agent merged the PR during the run, the ticket has
   already finished on merge (ADR 0021) and may sit in a done column. Does
   the play still apply move-to (dragging it back to In Review), or skip
   move-to when the ticket is already in a done stage? The owner wants a
   "merge-this" memory to let easy fixes go straight to done, so this
   ordering matters.
5. Failure and interrupt: the ticket stays, a system note lands in the
   thread, the button is available again. Confirm, and decide whether a
   failed run's partial Agent reply is still posted.
6. Provenance: the status change carries a new actor kind alongside `user`
   and `automation`, naming the play and the run. Confirm the name (`play`).
7. Events: `play.run_started`, `play.run_finished` (with outcome). Enough
   for automations to chain on, or is `play.run_failed` a separate topic?
8. Timeout: chat turns are bounded at ten minutes as a lost-signal guard.
   A fix-and-PR run may legitimately take longer. Per-play ceiling, or one
   longer instance-wide ceiling?

## Answer

Resolved 2026-09-16 with the owner (one grilling round).

- **States**: `starting` at the click, before the harness accepts; then
  `running`, `done`, `failed`, `interrupted`. A harness that refuses to
  start is a `failed` run with the reason.
- **Record**: play, target (ticket or doc), starter, selected memories,
  custom instructions, chosen move-to column, harness session id, started,
  ended, outcome, last error, the Agent's reply message id, and the
  harness's activity stream captured on the run, capped at a few hundred
  lines. The dialog reads the latest run to pre-select the user's last
  choices.
- **One active run per target**, whichever play; the button says a run is in
  progress. **Stop** is available to the starter and to anyone with
  `plays:write`; a stopped run keeps its record and logs intact, and
  continuing later is a new run in the same thread, whose harness session
  is reused.
- **Move-to never moves backwards.** The default automations already move
  a ticket to In review when a PR opens and act again on merge. On `done`
  the play applies the chosen column only if its stage is later than or
  equal to the ticket's current stage; otherwise it skips with a note in
  the thread. A chosen column deleted mid-run also skips with a note.
- **Failure and interrupt**: the ticket stays, any partial reply is posted,
  a system note says why it ended, and the starter gets an inbox
  notification for every terminal outcome so failures are reported, not
  just logged.
- **Events**: `play.run_started` and `play.run_finished` with an `outcome`
  field, so an automation can notify or chain. A status change made by a
  play carries actor kind `play` with the play's name and the run id.
- **Timeout is silence, not duration.** A run ends as `failed` when the
  harness has sent nothing (no snapshot, activity, or terminal) for a set
  time, on the order of fifteen minutes; there is no total ceiling, since
  a long run that keeps reporting is healthy. Chat's ten-minute total cap
  stays chat's.
