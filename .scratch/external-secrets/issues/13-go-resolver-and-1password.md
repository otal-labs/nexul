# 13 — The Go resolver interface, its 1Password implementation and a fake

**Status:** resolved

**Blocked by:** 11

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ADR 0181, the spec (The seam for a second manager, Measured facts), ticket 07.

## What to build

- `internal/platform/secretref` gains `Resolver` (scheme, `ResolveAll(ctx, refs)` returning a value
  or a typed error per reference: not found, field missing, no access, unreachable) and a registry that
  picks a resolver by scheme.
- A 1Password resolver on `github.com/1password/onepassword-sdk-go` pinned at `v0.4.1`: one client
  for the life of the process, `ResolveAll` in one call, SDK error types mapped to the typed errors
  (`vaultNotFound` → no access).
- An in-memory fake that can return every typed error, for every other ticket's tests.

## Acceptance criteria

- [ ] The fake and the error mapping are table-tested, including a batch with good and bad entries.
- [ ] A test asserts `cmd/nexul` does not import the SDK (it embeds a 9.5 MB wasm core); only
      `nexul-runner` and `nexul-server` may.
- [ ] `CGO_ENABLED=0` builds of the runner and server still succeed.

Verify: `make lint`, `go test ./internal/platform/...`, `make vuln`.

## Comments

Built: `secretref.Resolver` (results line up with the input slice), `Registry`, the typed errors, `secretref/onepassword` on the SDK at v0.4.1, and `secretref/secretreftest.Fake`. `cmd/nexul` is guarded against linking the SDK. Checked against real 1Password: a value, each typed error, and `ErrUnreachable` with the network cut.
