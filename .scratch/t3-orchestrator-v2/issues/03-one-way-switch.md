# 03 — A computer moves to the protocol-2 client on its first refusal, and never back

**What to build:** The owner's one-way move, spec decisions 2, 7 and 8.

`harness`: `Session.ComputerID` (fill it in `pairing.Computer.Session()`); `MovedError{To Kind}`;
`Forward(from, to Client, moved func(ctx context.Context, s Session, to Kind) error) Client`. For every
Session method, a `MovedError` from `from` calls `moved` once, then the same call on `to`; if
`moved` fails, return its error without calling `to`. `Pair` and `Version` have no session: on
`MovedError` they retry on `to` without calling `moved`. `Kind()` returns `from.Kind()`.

`t3client` — the three allowed protocol-1 changes:
1. `Pair` calls `t3rpc.Describe` BEFORE the token exchange: protocol 2 → `MovedError{To:
   KindT3CodeV2}` with the token unspent; above 2 → ErrProtocol "needs a newer Nexul". `Version`
   does the same from its descriptor.
2. Connect turns `t3rpc.ProtocolMismatchError` into `MovedError` (version 2) or ErrProtocol
   (above 2).
3. `reconnect` inside a running turn treats `MovedError`/ErrProtocol as final at once: Terminal error
   "T3 Code was updated during this turn; ask again".

`pairing`:
- The switch use-case runs `UPDATE pairing_computers SET kind='t3code-v2', harness_version=?,
  updated_at=? WHERE id=? AND kind='t3code' RETURNING user_id` (new sqlc query) with the outbox write
  in the same transaction; zero rows → no-op, no event. It re-reads `harness_version` from
  `t3rpc.Describe` first.
- `pair()` stores `PairResult.Kind` when non-empty, else `base.Kind`; refuses (ErrConflict, plain
  message) when `base.ID != ""` and the result's kind is lower than the stored one; writes
  `computer.harness_switched` when a re-pair raises the kind. Update the Repair doc comment ("the kind
  never changes").
- `requireSetup` passes an ErrProtocol through with its own message instead of ReasonOffline;
  `pairFailure` files an ErrProtocol under `server_url` with its own message, without "run t3 pair".
- The pre-turn version-drift warning compares against the switched computer's refreshed
  `harness_version`, so the first turn after a switch posts no "re-pair" warning.

`presence`: on ErrProtocol keep the capped retry (updating T3 recovers on its own) but log the refusal
once at Warn, then at Debug.

Event `computer.harness_switched` {computer_id, user_id, from_kind, to_kind, harness_version}: topic in
`internal/pairing/events.go` Topics(); schema in `internal/integrations/catalog.go`; `bun run
generate:events` in `sdk/`; `instanceScope` in `server/cmd/automation_scope.go`; the topic in the live
list in `server/cmd/main.go` and `ownFrame` in `server/cmd/live_audience.go`;
`"computer.harness_switched": [getComputersKey, getHarnessResolveKey]` in
`web/src/hooks/useLiveEvents.tsx`.

`server/cmd`: registry `{KindT3Code: harness.Forward(t3client, t3clientv2, switch), KindT3CodeV2:
t3clientv2}`, the switch bound late the way `presenceKeeper` is in `services.go`.

ADR 0113 (re-check the number against `origin/master`): two kinds for T3 Code and the one-way switch.
Amend `docs/adr/0054-one-harness-client-interface-per-kind.md` (two T3 kinds; the OpenCode 2 harness is
dropped because T3 runs OpenCode 2) and `docs/adr/0029-the-agent-turn-runs-on-the-mentioning-users-own-environment.md`
(Nexul refuses a T3 that went back). Update `CONTEXT.md` Harness.

**Blocked by:** 02

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`. Also `practices/mcp.md` (`computer_*` output shows the kind).

- [ ] Error paths first: `moved` failing returns its error and `to` is not called; `t3code-v2` → `t3code` re-pair is refused with ErrConflict; a 426 naming 3 → ErrProtocol; a turn whose reconnect hits the 426 ends at once with the "updated during this turn" error (reconnect_test)
- [ ] Forward table: each Session method → `moved` once, then `to`; Pair and Version → `to` without `moved`
- [ ] Fresh pair on a protocol-2 fake whose `/oauth/token` rejects a second use of a token → stored as `t3code-v2`
- [ ] Real-SQLite storage test of the switch query; two concurrent switches → one event
- [ ] requireSetup on a computer that went back returns the "went back" message, not "is T3 Code running there?"; the first turn after a switch posts no drift warning
- [ ] `internal/integrations/catalog_test.go` and `server/cmd/automation_scope_test.go` pass with the new topic; SDK regenerated; `make sqlc-check`, `make lint`, `make coverage` green
