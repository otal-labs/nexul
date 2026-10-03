# 02 — A protocol-2 client that connects, lists, and holds presence

**What to build:** New package `internal/t3clientv2` implementing `harness.Client` for
`harness.KindT3CodeV2` (`"t3code-v2"`), on `internal/t3rpc`.
- Dial with `Query{orchestrationProtocol: 2, clientAppVersion: nexul/<version>}`.
- `Pair` probes the descriptor first (never spend the one-time token on a protocol-1 server), then
  exchanges and sets `PairResult.Kind = KindT3CodeV2`. Add `Kind` to `harness.PairResult`;
  `t3client.Pair` sets `KindT3Code`.
- `Version`, `ListProviders`, `ListProjects`, `Hold` work against a protocol-2 server. `0.0.45` accepts a
  dial carrying `orchestrationProtocol=2` and speaks protocol 1 on it (ticket 04 Findings 11), so the
  refusal below must come from the handshake's protocol, never from the dial alone. `Providers()` sets
  `harness.Provider.Version` as protocol 1 does (ticket 15 Comments). The nightly's shell snapshot has
  `schemaVersion: 2` and a second snapshot with `resolvedRepositoryIdentityRoots`; project rows are
  unchanged.
- `t3rpctest` gains single-use pairing tokens and an exchange counter (ticket 01 Comments).
- Add `harness.ErrProtocol` (a sentinel wrapped with `%w`). A protocol-1 handshake or descriptor →
  ErrProtocol "T3 Code on <computer> went back to its old orchestrator; Nexul only moves forward.
  Update T3 Code there." (for `Pair`/`Version`, name the server URL's host). A 426 naming a protocol
  above 2 → ErrProtocol "T3 Code on <computer> needs a newer Nexul".
- `Settle` dispatches `thread.settle` with `commandId` and `threadId`, the protocol-1 shape (`settledAt` is
  optional and left out). T3 V2's sidebar hides handed-off child threads, so only the thread itself is settled.
- `StartTurn`, `Interrupt`, `Answer` return ErrInvalid "turns on this T3 Code version need a newer
  Nexul" until ticket 05 (Interrupt/Answer until 08).
- Register the kind in `server/cmd/services.go`; add `"t3code-v2": "T3 Code"` to `HARNESS_LABELS`
  in `web/src/models/Pairing.tsx`.
- Roadmap: remove the "Second agent harness (OpenCode 2)" bullet from `ROADMAP.md` (it exists only
  there) and add an in-progress item "T3 Code orchestrator V2" to both `ROADMAP.md` and
  `website/src/pages/roadmap.astro`.

**Blocked by:** 01

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`, ticket 04's Findings and ticket 01's Comments.

- [ ] Error paths first: protocol-1 handshake → `errors.Is(err, harness.ErrProtocol)`, not Retryable, message names the computer; 426 naming 3 → ErrProtocol; ticket mint 401 → ErrUnauthorized
- [ ] `Pair` on a protocol-1 descriptor never calls `/oauth/token` (the fake counts exchanges)
- [ ] Against the protocol-2 fake, ListProviders, ListProjects and Hold return the same harness values protocol 1 produces; the dial URL carries both params
- [ ] Website builds; `bun run lint` and `bun run typecheck` in `web/`; `make lint`, `make coverage` green
