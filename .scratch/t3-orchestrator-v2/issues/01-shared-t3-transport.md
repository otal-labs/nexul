# 01 — Move T3's protocol-neutral transport into its own package

**What to build:** A refactor with no behavior change. The protocol-neutral parts of
`internal/t3client` move to `internal/t3rpc`, so the protocol-1 client and the coming protocol-2
client share one transport. Exported API (adjust names only if the code makes one clumsy):

- `Connect(ctx, harness.Session, Options{HTTPClient, Logger, RPCTimeout, Query url.Values}) (*Conn, error)`
  — mints the ticket, dials `/ws` with `Query` plus `wsTicket`, runs the `server.getConfig`
  handshake. Keeps the dial response: a 426 returns `*ProtocolMismatchError{Version int}` read from
  the body's `orchestrationProtocolVersion`.
- `(*Conn).Call(ctx, method, payload)`, `(*Conn).Stream(ctx, method, payload)` (a stream that hands
  each Chunk's values to the caller, acks after them as today, and sends Interrupt on Close),
  `Done()`, `Err()`, `Config()`, `Protocol()` (from the handshake's `environment`; absent → 1),
  `Providers()`, `ListProjects()` (`projects.go`, the `subscribeShell` snapshot, moves here).
- `Exchange(ctx, *http.Client, serverURL, token)` and `Describe(ctx, *http.Client, serverURL)
  (Descriptor{ServerVersion string; Protocol int}, error)` for `/.well-known/t3/environment`.
- The default-model pick (`defaultModelFor` today) moves here too; both clients need it.

`t3client` keeps `thread.go` and `harness.go`, wrapping `type conn struct{ *t3rpc.Conn }` for its
protocol-1 methods. The fake T3 server moves from `fake_test.go` to an importable, coverage-exempt
`internal/t3rpc/t3rpctest` (like `internal/harness/harnesstest`) so `t3rpc`, `t3client` and
`t3clientv2` tests share it. Fix the stale "pinned to v0.0.34" package comment.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, `practices/architecture.md`, the spec, and `research/protocol-2-wire.md`.

- [ ] The same test scenarios and assertions pass; only receivers, constructors and imports change
- [ ] Table test: descriptor and getConfig with protocol absent → 1, 1 → 1, 2 → 2; a 426 body naming 2 → `ProtocolMismatchError{Version: 2}`
- [ ] `t3rpctest` is listed as coverage-exempt the way `testutil/` is
- [ ] `make lint`, `make vet`, `make coverage` green
