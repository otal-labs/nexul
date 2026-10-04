# 23 — Follow T3's restart recovery: a cut run waits for its handed-off work, and a Watch adopts the run T3 ran last

Found by the T3 drift check of 2026-10-04 (T3 31a9da179e → eac52f0087, nightlies 2638 and 2644): nothing on the wire changed, but T3 5108c978b1 (#15323) changed restart recovery in two ways Nexul must follow.

**What to build:** Two fixes in `internal/t3clientv2` for T3's new restart recovery (T3 5108c978b1), plus the matching doc and tracker edits. Re-read the code first; line numbers drift.

1. **A run that a T3 restart cut waits like a done run.** In `internal/t3clientv2/watch.go`:
   - Add `func (w *watch) cut() bool { return w.run.Status == "cancelled" && w.thread.DeletedAt == nil }`.
     - Stop yields "interrupted". `queued-run.cancel` and archive cancel only unstarted runs (T3 Orchestrator.ts:3139-3153, 7157, 7360), and those have no followed runs or subagents, so `pending()` is already false for them.
     - Deletion emits `thread.deleted` before the cancel (T3 ThreadDeletion.ts:57-83), so `DeletedAt` is set by the time the cancel arrives.
   - `end()`: when `cut()`, treat the run as done. `steered()` and `pending()` apply exactly as they do for a done run. Once nothing is pending, end `TurnDone` if any followed run other than `w.run.ID` reached `runWaiting` or `runCompleted`; otherwise end `TurnInterrupted`, as today.
   - `waiting()`: add `|| w.cut()`, so `pump.capped` (pump.go:182-194) applies the 60-minute cap and ends the turn `TurnDone` with `LeftRunning`.
2. **A Watch adopts the run T3 ran last.** Match T3's `runRanAfter`:
   - In `watch.go`, add `CompletedAt *string` with JSON tag `completedAt` to `run`.
   - In `turn.go` `adopted()`, take as newest the unfinished run (status in `liveRuns`) with the highest ordinal. If no run is unfinished, take the run with the latest `completedAt`, parsed with `time.RFC3339Nano`; on a tie, the higher ordinal wins.
   - Look for chain roots only in `p.Runs[:i]`, where `i` is newest's index. Return `newest.UserMessageID` when no earlier chain leads to it.
   - Leave `stop.go` (`halt`, `toStop`, `interruptChild`) as it is. `halt`'s `runs[len-1]` mirrors T3's own `runs.at(-1)` check (Orchestrator.ts:8030), and the live-run picks rank only unfinished runs, where ordinal order is correct.
3. **A held wake gets the held note.** If a followed wake run is queued with `queueHeld`, `standingNote()` in
   `watch.go` still says "Waiting for work handed off in T3 Code" for up to an hour. After the `pending()` guard,
   return `heldNote` under the existing call id `"handoff:"+w.run.ID`, and add one clause to ADR 0116's "While it
   waits". With restart continuation on, a post-restart wake that queues behind the continuation inherits
   `queueHeld`, which is exactly the window item 1 makes the turn wait through.
4. **Docs and tracker:**
   - ADR 0116, "When it is over": a run cancelled by a T3 restart waits for its handed-off work like a done run. Only a Stop, a failure, or a deleted thread ends the turn at once.
   - `research/protocol-2-wire.md:510`: restart recovery's "cancelled" waits for handed-off work. Only a queued-run cancel maps straight to interrupted.
   - `research/protocol-2-wire.md:237`: since T3 88744f3ddb, an agent-only thread (a handed-off child) is windowed too, to at most 75 items. It used to get its whole history.
   - `research/runs-nexul-did-not-start.md:23`: `settled_only` holds back while the *spawning run* is live (Orchestrator.ts:8699-8705), not while any run on the parent is live.
   - Ticket 17: Pi 0.99 is Nexul's setup floor (its MCP client). T3's own floor is Pi 0.80.5, with 1.0 recommended (pingdotgg/t3code#14871).

Known limits, not in scope:
- Nexul resubscribes about 31 s in total (1, 5 and 25 s, pump.go:31), so a slower T3 restart still ends the turn with the connection-lost error.
- A cut run with no handed-off work, whose opt-in continuation arrives after the cancel is processed, is still missed.
- A Watch on the snapshot `[R0 cancelled, R1 held, R_c running]` still adopts R0's chain. Watch has no message id to do better.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `.scratch/t3-orchestrator-v2/research/protocol-2-wire.md` (run status mapping, bounded snapshots), `.scratch/t3-orchestrator-v2/research/runs-nexul-did-not-start.md`, ADR 0114, ADR 0116, the Comments on ticket 11.

- [ ] `watch_test.go`, rows in the `TestWatch_HandedOffWork` table:
  - A restart-cancelled own run with an app_owned task still running emits no terminal. The wake run (message `delegatedCompletion.parentRunId` = own run) then replies and ends the turn "end done" with that reply.
  - A cut run with nothing pending ends "end interrupted" at once.
  - A cut run on a deleted thread ends interrupted at once, even with work pending.
  - A cut run whose child ends with no followed run reaching waiting or completed ends interrupted.
- [ ] `pump_test.go`: a cut run with pending work repeats `handoffNote`, and the cap ends it done with `LeftRunning` (next to `TestPump_HandedOffWorkStillRunning_StepRepeatsUntilTheCapEndsTheTurnDone`).
- [ ] `turn_test.go`:
  - Watch on `[R0 cancelled, R1 running, R_c completed with restartContinuationOfRunId=R0]` follows R1's message.
  - Watch on `[R0 cancelled, R_c completed, R1 completed later]` ends at once with R1's reply.
  - `TestWatch_NewestRunIsAWakeOfAnEarlierRun_FollowsTheRunThatHandedOff` stays green.
- [ ] A followed wake run held with `queueHeld` shows the held note under the hand-off call id (watch_test row)
- [ ] `go test ./internal/t3clientv2/...`, `make lint` and `make coverage` are green.
- [ ] ADR 0116, both research files and ticket 17 are updated in the same PR. Hit every surface: only the harness client and docs apply (no route, MCP tool, event, push or UI).
