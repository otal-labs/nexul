# 12 — Stop also stops the work a turn handed off

**What to build:** Spec decision 13. Stop on a protocol-2 turn, in this order:
1. For each `app_owned` subagent of a followed run whose `completionDelivery.state` is not
   `acknowledged`, `delivered` or `disposed`: `{"type":"delegated_task.completion-delivery.dispose",
   "commandId":…,"parentThreadId":<thread>,"taskId":<subagent id>}` — first, so a child's end cannot
   wake the parent.
2. For each followed subagent that is pending, running or waiting with a child thread:
   `getThreadProjection` on the child, then `run.interrupt` its latest preparing, starting or running
   run. A failure is logged and noted, and does not stop the rest.
3. A followed wake run: queued → `queued-run.cancel`; live → `run.interrupt`.
4. Nexul's own run as in ticket 08 (a waiting latest run with background work is interruptible; "not
   interruptible" counts as done).

The turn ends interrupted, and no wake run is followed afterwards.

**Blocked by:** 08, 11

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`.

- [x] Stop while waiting on an async child: dispose sent before the child's interrupt, the child's run interrupted, the turn interrupted, a later wake ignored
- [x] A child interrupt failure is noted and the parent's interrupt still goes out; a delivery already disposed sends no dispose

## Comments

- **Where it lives.** `stop.go`: `live` is what Stop and the turn share (the message id, the hand-offs the pump
  publishes, the notes Stop leaves, and `halt`); `Harness.turns` (thread id to `*live`) replaced `Harness.messages`.
  `stopHandoffs` is steps 1 to 3, `halt` is the one cancel-or-interrupt rule for Nexul's run, wake runs and child runs,
  `interruptChild` reads the child thread first. `pump.fold` publishes `watch.handoffs()` after each batch;
  `pump.finish` sends Stop's notes before any Terminal; `drain` ends the turn with `watch.stop()` once `halt` runs.
- **Fake server.** `t3rpctest.Server.Projections` answers `getThreadProjection` per thread id and replaces
  `Projection`; a thread it does not name fails the way T3 fails a thread it cannot load. `AfterCommand` swaps in a
  thread's projection once a command lands, as T3's own follow-up events would.
- **Judgment calls.**
  - The followed set comes from the turn's own watch, published by the pump, not rebuilt from `getThreadProjection`:
    the projection's window can drop the link messages and a finished task whose result is still undelivered. Run
    statuses for steps 3 and 4 come from a projection read after steps 1 and 2, which holds every live run: dropping
    the last result of a queued wake cancels that wake in T3, and T3 refuses to cancel it again ("is not queued").
    That second read is taken only when the turn handed off work, and its failure fails Stop like the first read's.
  - Once Stop has stopped anything, the turn ends interrupted at once, without waiting for T3's own report: an
    interrupted waiting run with background work can stay waiting and then complete, and a dropped result reports no
    end. This also applies to a plain Stop with nothing handed off, so text T3 streams after Stop is not shown.
  - Stop succeeds when a hand-off step stopped something even if Nexul's own run is already over; with nothing stopped
    anywhere it is still `ErrConflict`. A refusal on Nexul's own run still fails Stop and does not end the turn.
  - Every hand-off step's failure (dispose, the child's read or interrupt, a wake run's cancel or interrupt) is treated
    like the child failure the ticket names: a Warn log and the note "Could not stop handed-off work in T3 Code:
    <error>", shown just before the turn's Terminal, also when T3 ends the turn first.
  - A T3-owned task with no `completionDelivery` yet counts as undelivered and is disposed; T3 sets the state only
    when the task ends, and disposing it while it runs is what keeps that end from waking the thread.
  - A wake run goes through the same `halt` as Nexul's run: a waiting wake run is interrupted only as the thread's
    latest run, and "not interruptible" counts as done.
  - A provider's own subagent has no runs on its child thread, so step 2 finds nothing there; interrupting the parent
    (T3 settles its background work) is what stops it.
  - Known limit, from ticket 08: `Harness.turns` holds one turn per thread, so with two turns live on one T3 thread,
    Stop reaches the newer turn's hand-offs and run. The pipeline still calls Interrupt once per turn.
- **Ticket 13.** A stopped turn ends through `watch.stop()` without seeing its children's last state; a pill still
  `running` at that point should become `interrupted`.
- **Ticket 16.** Check live that T3 accepts `delegated_task.completion-delivery.dispose` on a running task from a
  paired client, that the child run's interrupt lands, and that no wake run starts after Stop.
- **Tests.** `TestInterrupt_TurnWaitingOnHandedOffWork_StopsItBeforeItsOwnRunAndEndsInterrupted` drives a real turn
  through the fake: an async child (dispose, then the child's interrupt, then the parent's), a child that cannot be
  read (noted, the parent still interrupted, an already disposed result not disposed again), and a queued wake after
  Nexul's run completed that T3 cancels with the dropped result (dispose only, no note, Stop succeeds, though T3
  refuses a second cancel). Each case ends interrupted with the update stream closed. On ticket 08's `Interrupt` all
  three fail (two never end, one is `ErrConflict`); deciding step 3 from the read before the dispose fails the third
  with a false "Could not stop handed-off work" note. A fourth case fails the read after steps 1 and 2: Stop fails
  before Nexul's own run and the turn runs on.
- **ADR.** ADR 0116 gained a Stop bullet; ADR 0114's Stop bullet points at it.
