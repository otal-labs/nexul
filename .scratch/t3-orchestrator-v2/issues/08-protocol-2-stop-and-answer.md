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

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [x] Error paths first: a rejected respond returns T3's message; a projection failure returns an Interrupt error; nothing live → ErrConflict and the plays Stop still closes the trail; "not interruptible" on a waiting run is not an error
- [x] Interrupt table: running → no `holdQueue` key; queued → cancel; unknown message id → newest non-terminal run
- [x] `TestStartTurn_ContextEndsWhileQueued_CancelsOwnRun`; answer-encoding table; the setup turn's interrupt-on-question still works

## Comments

- **Where it lives.** `stop.go` is `Interrupt`: it reads `orchestration.getThreadProjection` (`readProjection`), picks
  the run (`toStop`) and stops it (`stopRun`). `answer.go` is `Answer` and `encodeAnswers`. Ticket 12 runs its steps 1 to
  3 before `stopRun`, which is its step 4 unchanged.
- **The message id.** `Harness.messages` maps a thread id to the message id of the turn watching it. `StartTurn` stores
  it once the message is dispatched; `turn.end` removes it with `CompareAndDelete` when the pump stops, so a newer turn
  on the same thread keeps its own entry.
- **Projection.** `projection.RuntimeRequests` now decodes each request's `id` and `responseCapability.type`. Ticket 09
  adds `status` for its resolved, expired and not-resumable checks, and can call `encodeAnswers(answer, asMessage)` for
  its respond.
- **`t3Message(err)`** is the message T3 refused a call with; `refused` builds on it.
- **Fake server.** `t3rpctest.Server.Projection` answers `getThreadProjection`, and `ProjectionCause` fails it. A command
  failed through `CommandCauses` is not recorded on `Dispatched`, so a refused command is seen only through the error it
  returns.
- **Judgment calls.**
  - When the watched turn's own run is in the projection but already finished, Stop is `ErrConflict`; it does not fall
    back to another run on the thread, which would stop work Nexul did not start. The fallback to the newest unfinished
    run applies only when no turn is watching or T3 has no run for the message id.
  - "Not interruptible" counts as done only for a `waiting` run. On a live run it is T3's refusal and comes back as an
    error. A `waiting` run with a later run after it is `ErrConflict`, since only the latest run can have background work.
  - The queued run is cancelled whenever the pump stops while the watch still sees it queued: the turn's context ending,
    and also three failed resubscribes or a protocol change, which spec decision 13 calls a turn giving up.
  - Answers pass the chosen values through as the caller sends them. The web (`optionValue`) and the `trail_update` MCP
    schema already send an option's value, else its label, so ticket 07's question mapping must fill
    `QuestionOption.Value` from T3's `option.value`.
  - A live answer mirrors protocol 1: free text wins, one choice is a string even on a multi-select question, several
    are a list. T3's own web sends a list for any multi-select; live Claude joins lists and live Codex takes either.
  - `Answer` reads the projection to learn the request's capability. A request missing from it is answered live-style
    and T3's reply decides. "Runtime request <id> is resolved." becomes `ErrConflict`, which chat and plays already treat
    as answered; expired, cancelled, not found and not resumable come back as `ErrInvalid` with T3's message.
  - Chat's Stop with nothing to stop now answers 409 "nothing is running on this T3 thread"; protocol 1 succeeded
    silently.
- **Tests.** The plays half of "nothing live → ErrConflict" is `TestStop_HarnessInterruptFails_ClosesTheTrailHere`,
  which closes the trail on any interrupt error, so no copy of it was added. The setup turn's interrupt-on-question is
  `TestInterrupt_AfterStartTurn_StopsThatTurnsRunNotTheNewest`: a live question keeps the run `running`, and Stop finds it
  through the message id `StartTurn` stored. `Interrupt` and `Answer` joined the session-call tables, so both refuse a
  T3 that went back with the plain message.
- **ADR.** ADR 0114 gained a Stop and an Answers bullet.
