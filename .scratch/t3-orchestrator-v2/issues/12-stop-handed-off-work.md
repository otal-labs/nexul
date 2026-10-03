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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md` and `research/runs-nexul-did-not-start.md`.

- [ ] Stop while waiting on an async child: dispose sent before the child's interrupt, the child's run interrupted, the turn interrupted, a later wake ignored
- [ ] A child interrupt failure is noted and the parent's interrupt still goes out; a delivery already disposed sends no dispose
