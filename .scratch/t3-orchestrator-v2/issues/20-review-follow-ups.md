# 20 — Follow-ups from the protocol-2 reviews

**What to build:** The small defects the reviews of tickets 02–12 found, in one PR. Each item names
where it is; re-read the code first, since line numbers drift.

1. **Data race on master.** `internal/t3client/harness.go` `reconnect` reads `w.lastSeq` (and logs it in
   `thread.go`'s "connection lost mid-turn" Warn) while the resumed subscription goroutine writes it.
   `TestHarness_ConnectionDropsMidTurn_ResumesFromTheLastSequenceWithoutGapsOrDuplicates` races about once
   in 600 runs under `-race`. Make the read safe (pass the value, or guard it) without changing protocol-1
   behavior; prove it with the test at `-race -count=2000`.
2. **Hand-off delivery state survives partial updates.** `internal/t3clientv2/watch.go` replaces a subagent
   row wholesale on `subagent.updated`; T3 keeps a known `completionDelivery` when an update omits it
   (`apps/server/src/orchestration-v2/ProjectionStore.ts` `preserveCompletionDelivery`). Do the same, and
   drop the run `delegatedCompletion` carry-forward that nothing reads. Fix the comment that overstates what
   a bounded snapshot keeps.
3. **Stop tells the truth.** In `internal/t3clientv2/stop.go`: when every hand-off step failed and Nexul's
   own run is over, return the first hand-off error, not "nothing is running"; build notes from the T3
   message without the `apperrs` prefix; do not repeat notes on a second Stop; when Nexul's own run was read
   as queued or running and T3 refuses with "is not queued" or "not interruptible", re-read once and act on
   the fresh status. Rename the `live` type so it does not clash with `(*turn).live`.
4. **One respond shape.** `runtime-request.respond` is encoded by two structs (`answer.go` and `turn.go`);
   merge them into one with `decision` and `answers` both `omitempty`. `turn.decline` uses `t.live(ctx)`
   like the rest of the file.
5. **Answer fallbacks.** `answer.go`: when the request's run cannot be found in the snapshot, dispatch the
   answer as text instead of following an empty message id; fix the comment that says it returns an id.
6. **Untested guard.** Add the Interrupt row the review asked for: after a turn's watch ended, Stop takes the
   newest-unfinished fallback (fails if `CompareAndDelete` is removed).
7. **A switch refreshes presence.** `pairing.SwitchHarness` calls `notifyComputersChanged(userID)` when a row
   changed, so a presence loop started under the old kind restarts on the new one.
8. **Words that are wrong.** `harness.Session.ComputerID` doc ("names the paired computer, so Forward's moved
   callback knows which row to switch"); `internal/t3clientv2/images.go` reason ("the types T3's providers take
   natively; Claude fails the run on any other"); `CONTEXT.md` model options sentence from ticket 18 (same
   provider and no model keeps the thread's model; same model and no options keeps its options; another
   model starts on its defaults); ADR 0113 gains a "see ADR 0114" line for the protocol-2 turn; ADR 0114's
   Stop bullet says a successful protocol-2 Stop ends the turn interrupted at once; the spec's Docs line
   names 0114 and 0116 for what they hold.

**Blocked by:** None — can start immediately

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`.

- [x] The race test passes `-race -count=2000`
- [x] A `subagent.updated` without `completionDelivery` keeps the pending state (test)
- [x] Stop: all hand-off steps failing returns the first error; a second Stop adds no notes; a refused cancel or interrupt on Nexul's own run re-reads once (tests)
- [x] One respond struct; the Interrupt fallback row; a switch calls `notifyComputersChanged` once (test)
- [x] `make lint`, `make vet`, `make coverage` green; protocol-1 scenarios unchanged

## Comments

- **1, the race.** Already fixed on master by #366, which reads the sequence before the resumed subscription starts;
  this ticket's review predates it. Putting the old read back reproduces the report (`reconnect` against
  `eventUpdate`, about once in 8,000 runs here), and master passes the named test at `-race -count=2000` (and 12,000
  runs across six processes). `internal/t3client` is unchanged, so protocol 1 behaves exactly as before.
- **2, hand-off state.** `watch.subagent` keeps a row's known `completionDelivery` when `subagent.updated` leaves it
  out. The run `delegatedCompletion` carry-forward is gone: `deliveries` is never cleared, so an update without one
  already keeps the mapping. A resubscribe snapshot still replaces the subagent rows (spec decision 9), and a bounded
  snapshot keeps a finished subagent only while its run is in the window. Ticket 13 should keep a pill's last known
  state rather than reading it back from `w.subagents` after a snapshot.
- **3, Stop.**
  - `stopHandoffs` returns the hand-off outcome as an error (a `tally`): nil once a step stopped something, else the
    first failure, else `errNothingToStop`. `Interrupt` uses it only when Nexul's own run had nothing to stop.
  - Each Stop replaces the turn's notes with its own failures, so a second Stop neither repeats a note nor keeps one
    for work it then stopped. A note keeps the step's words and T3's message, without the `invalid:` prefix ("Could not
    stop handed-off work in T3 Code: read the T3 thread: Failed to load orchestration V2 thread th-2").
  - `movedOn` marks "is not queued." on a cancel and "is not interruptible." on an interrupt of a run read as live.
    `stopRun` then reads the thread once more and acts on the fresh status. It applies to the run Stop is for, also
    when no turn is watching. Wake and child runs are not re-read; their refusal is a noted failure as before.
  - `live` is now `runningTurn` (`newRunningTurn`); `Harness.turns` holds `*runningTurn`.
- **4, one respond shape.** `requestRespond` in `answer.go` carries `decision` and `answers`, both `omitempty`;
  `turn.decline` goes through `t.live(ctx)`. The existing payload tests pin both shapes.
- **5, answer fallback.** `deliver` works out the run to follow before responding; a live request whose run the
  snapshot does not hold is sent as a message, so nothing is answered twice.
- **6, the fallback after the watch ended.** A separate test, not a table row: the table seeds `Harness.turns`
  directly, and only a real turn ending runs `CompareAndDelete`. The test ends a queued turn through its context, so
  T3's `queued-run.cancel` (sent after the delete) is the signal to wait on. With the delete removed, Stop answers
  "nothing is running" and the test fails.
- **7, presence.** Judgment call: the notification alone did not restart anything, because the keeper's `reconcile`
  kept any running loop by computer id. A loop now remembers its kind, and `reconcile` replaces one whose computer's
  kind changed. Without that, a loop started under `t3code` kept holding through `Forward`'s protocol-1 client and
  showed a computer that went back to stable as connected. A re-pair that raises the kind gets the same restart.
- **8, docs.** CONTEXT.md's Model options entry, the `Session.ComputerID` and `supportedImages` comments, ADR 0113's
  pointer to 0114 (and the presence restart), ADR 0114's Stop bullet (Stop ends the turn at once, and the re-read) and
  its answer fallback, ADR 0116's Stop bullet (first refusal, notes replaced) and `completionDelivery` kept, and the
  spec's Docs list.
- **Fake server.** `t3rpctest.Server.AfterCommand` now applies when the command arrives, refused or taken, so a test
  can model a run that moved on before the command landed.
