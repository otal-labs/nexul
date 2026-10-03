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

**Status:** done

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, `research/protocol-2-wire.md`, ticket 04's Findings and ticket 01's Comments.

- [x] Error paths first: protocol-1 handshake → `errors.Is(err, harness.ErrProtocol)`, not Retryable, message names the computer; 426 naming 3 → ErrProtocol; ticket mint 401 → ErrUnauthorized
- [x] `Pair` on a protocol-1 descriptor never calls `/oauth/token` (the fake counts exchanges)
- [x] Against the protocol-2 fake, ListProviders, ListProjects and Hold return the same harness values protocol 1 produces; the dial URL carries both params
- [x] Website builds; `bun run lint` and `bun run typecheck` in `web/`; `make lint`, `make coverage` green

## Comments

- `harness.ErrProtocol` is matched with `errors.Is`, but a refusal is built with `harness.ProtocolRefusal(msg)`
  rather than `fmt.Errorf("%w: …")`, so its `Error()` is the message alone, with no sentinel prefix. The spec has
  everything downstream show the refusal as is, so ticket 03 shows `err.Error()` in `requireSetup` and
  `pairFailure` with nothing to strip.
- The refusal text is exact: "T3 Code on <computer> went back to its old orchestrator; Nexul only moves forward.
  Update T3 Code there." and "T3 Code on <computer> needs a newer Nexul." `<computer>` is `Session.Name`, else
  the server URL's host with its port; `Pair` and `Version` always use the host. Nothing wraps it as Retryable.
- `t3clientv2` refuses on the handshake's protocol after the dial (0.0.45 accepts the protocol-2 dial). Any 426
  answering a protocol-2 dial reads as "needs a newer Nexul", since only a server past protocol 2 sends one.
  A descriptor or handshake naming 3 or above gives the same message.
- `Pair` reads the descriptor through `Version`, so a refused protocol never reaches `/oauth/token`. A
  `Describe` failure comes back wrapped as "read T3 version", still Retryable when the server is unreachable.
- The dial carries `orchestrationProtocol=2` and `clientAppVersion=nexul/<version.Version>` (`nexul/dev` in a
  local build), and no `clientSurface`.
- `t3rpctest.Server` now has `PairToken` (default `pair-token`, accepted once by `/oauth/token`, then 400
  `invalid_grant`), `Exchanges` (every `/oauth/token` request), and `Dialed` (each `/ws` query without the
  ticket, refused dials included). From `Protocol` 2 the shell answers with `schemaVersion: 2` and a second
  snapshot carrying `resolvedRepositoryIdentityRoots`. `getConfig` now lists one installed Claude provider
  (`2.1.288`, one default model), and the `"ok": true` key no test read is gone.
- `StartTurn`, `Interrupt` and `Answer` return ErrInvalid "turns on this T3 Code version need a newer Nexul"
  without dialing. Tickets 05 and 08 replace them.
- Registering `t3code-v2` lets the pairing handler accept `kind: "t3code-v2"` today, because it looks kinds up in
  the registry. Such a computer pairs, lists, holds presence and settles, and its turns get the error above.
  Ticket 03 swaps the `t3code` entry for `harness.Forward`; both registry entries share one `t3rpc.Options`.
- `practices/go.md` and `practices/architecture.md` list `t3clientv2`. ADR 0054, ADR 0029 and `CONTEXT.md` are
  still ticket 03's.
