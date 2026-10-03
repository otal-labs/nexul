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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`.

- [ ] The race test passes `-race -count=2000`
- [ ] A `subagent.updated` without `completionDelivery` keeps the pending state (test)
- [ ] Stop: all hand-off steps failing returns the first error; a second Stop adds no notes; a refused cancel or interrupt on Nexul's own run re-reads once (tests)
- [ ] One respond struct; the Interrupt fallback row; a switch calls `notifyComputersChanged` once (test)
- [ ] `make lint`, `make vet`, `make coverage` green; protocol-1 scenarios unchanged
