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

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, the spec, ADR 0116, ticket 23's ticket file and Comments.

- [ ] Own run waiting, then restart-cancelled, child ends with no woken reply: the turn ends done with its own reply (watch_test row, fails on master)
- [ ] Cut path, woken run arrives completed without waiting: ends done with that reply (watch_test row)
- [ ] Spec and comments updated; `make lint`, `make vet`, `make coverage` green; protocol 1 untouched
