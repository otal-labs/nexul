# 16 — A runner checks its token at start and reports its managers' states

**Status:** ready-for-agent

**Blocked by:** 13, 15

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/mcp.md` (section 7), ADRs 0044, 0137, 0181, the spec (Runners, At start;
The seam for a second manager).

## What to build

- At start, a runner with a token opens its resolver's client (no read) and keeps it.
- The connect query (`internal/runner/client.go`, beside `version`) reports the managers and their states
  (`ready`, `token rejected`), named per manager, never a 1Password-specific field.
- The server stores the state on the runner and adds it, as a new field beside the existing ones, to
  `runner.connected` (`make event-schemas`) and to MCP `machine_list`.

## Acceptance criteria

- [ ] With the fake resolver: a good token reports `ready`, a rejected one `token rejected`, and no
      token reports no manager.
- [ ] `runner.connected` and `machine_list` carry the list; the event contract test passes.
- [ ] Personal runners never report one.

Verify: `make lint`, `make event-schemas`, `go test ./internal/runner/... ./server/...`.
