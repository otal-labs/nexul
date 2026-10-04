# 24 — A restart-cut run that already replied ends done, and the spec follows ticket 23

**What to build:** The follow-ups the review of ticket 23 (#416) found. Re-read the code first.

1. **A reply survives a restart cut.** T3's restart recovery also cancels a `waiting` run that has no pending
   checkpoint effect (`apps/server/src/orchestration-v2/ProviderRuntimeRecoveryService.ts:212-234, 336-345` at
   T3 eac52f0087). Then `cut()` in `internal/t3clientv2/watch.go` hides that Nexul's own run had already
   replied, and if no woken run replies, the turn ends interrupted where it would end done without the
   restart. Remember that the turn's own run reached `waiting` (or `completed`), and in `end()` count that as
   a reply, so the turn ends `TurnDone` with its own reply.
2. **Test gap.** Add a `TestWatch_HandedOffWork` row where, on the cut path, a woken run arrives as `completed`
   with no `waiting` first (a resubscribe answered with a snapshot); it must end done with that reply. This
   catches `replied()` narrowed to `runWaiting`.
3. **Words.** `internal/t3clientv2/pump.go` comments on `handoffCap`, `capAt` and `capped` cover a cut run as
   well as a done one. In `.scratch/t3-orchestrator-v2/spec.md`: decision 9 says a run cancelled by a T3
   restart waits for its handed-off work (only Stop, a failure or a deleted thread end it at once); decision 15
   says the cap counts from `waiting` or from the restart cut, and a held wake shows the held note.

**Blocked by:** None — can start immediately

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, the spec, ADR 0116, ticket 23's ticket file and Comments.

- [x] Own run waiting, then restart-cancelled, child ends with no woken reply: the turn ends done with its own reply (watch_test row, fails on master)
- [x] Cut path, woken run arrives completed without waiting: ends done with that reply (watch_test row)
- [x] Spec and comments updated; `make lint`, `make vet`, `make coverage` green; protocol 1 untouched

## Comments

- **Where it lives.** `t3clientv2/watch.go`: `watch.ownReplied` is set in `own()` once the turn's own run is seen
  `waiting` or `completed` and is never cleared, so it survives the restart's cancel and any later snapshot. `end()`
  ends a cut turn interrupted only when neither `ownReplied` nor `woken(replied)` holds. The reply itself needs no
  change: the own run's text was already emitted, and nothing after the cut replaces it unless a woken run streams.
- **Tests.** Two `TestWatch_HandedOffWork` rows. "a run that replied before a T3 restart cut it ends done on its own
  reply when no woken run replies" fails on the code before this change (`end interrupted`). "a cut run's wake seen
  only once completed, its waiting missed, ends done on that reply" passes before and after, and fails when
  `replied()` is narrowed to `runWaiting`; no other test catches that.
- **Judgment calls.**
  - The completed-without-waiting row drops the wake's `waiting` update from the event stream instead of sending a
    snapshot, the way `TestWatch_RecordedTurns/done at completed when waiting was missed` models a missed `waiting`.
    `reset()` reaches the same `end()`, and the table stays events only.
  - In the regression row the woken run fails rather than its result being disposed, so the child's work does end
    and only the woken reply is missing, mirroring ticket 23's "ends interrupted" row with `waiting` reached first.
  - ADR 0116's "When it is over" said a cut turn is done only "if a run it woke replied"; it now names Nexul's own
    reply before the restart too. The research file's status mapping and ADR 0114's pointer stay accurate as they are.
- **Still open.** Ticket 23's three known limits stand. A Watch that starts on a snapshot where its run is already
  cut never saw `waiting`, so it still ends interrupted when no woken run replies.
