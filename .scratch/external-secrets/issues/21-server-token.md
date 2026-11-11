# 21 — The server's own 1Password token, from install or a command, checked at start

**Status:** ready-for-agent

**Blocked by:** 13, 15

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, ADRs 0020, 0181, 0183, the spec (The server).

## What to build

- `nexul install --1password-token-file F` and `nexul server 1password` (stdin, `--remove`) write the
  token into the server's 0600 `.env` and restart it; `upgrade` keeps it (`recordVersion` rewrites
  that file).
- The server opens its resolver at start and shows its manager's state in Settings (an instance-wide
  read for whoever sees connectors).

## Acceptance criteria

- [ ] Install fakes cover set, rotate, remove and an upgrade that keeps the token.
- [ ] A rejected token leaves the server running, with `token rejected` in Settings.
- [ ] `NEXUL_AUTH_SECRET=op://…` is refused at start with a message naming ADR 0183's reason.

Verify: `make lint`, `go test ./internal/install/... ./internal/platform/config/... ./server/...`.
