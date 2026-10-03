# 08 — Stop and answer on protocol 2

**What to build:** `t3clientv2.Interrupt` reads `orchestration.getThreadProjection` and picks the run
whose `userMessageId` is the thread's last Nexul message id (a `sync.Map`, cleared with
CompareAndDelete when the watch ends), else the newest non-terminal run: `queued` → `queued-run.cancel`;
preparing/starting/running → `run.interrupt` with reason "Stopped from Nexul" and no `holdQueue`;
`waiting` as the thread's latest run → `run.interrupt`, treating "not interruptible" as done; nothing →
`ErrConflict`. Ticket 12 adds handed-off work. When a turn's context ends while its run is still
queued, cancel it on a detached 10-second context. `Answer` sends `runtime-request.respond` keyed by
question id, option value falling back to its label; a request whose `responseCapability` is
`message` needs one non-empty string per question, multi-select joined with ", ". A message-mode
question is followed by Terminal at `waiting`, which plays already treat as "waiting for an answer".

**Blocked by:** 05

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [ ] Error paths first: a rejected respond returns T3's message; a projection failure returns an Interrupt error; nothing live → ErrConflict and the plays Stop still closes the trail; "not interruptible" on a waiting run is not an error
- [ ] Interrupt table: running → no `holdQueue` key; queued → cancel; unknown message id → newest non-terminal run
- [ ] `TestStartTurn_ContextEndsWhileQueued_CancelsOwnRun`; answer-encoding table; the setup turn's interrupt-on-question still works
