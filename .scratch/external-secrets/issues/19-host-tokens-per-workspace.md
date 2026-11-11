# 19 — An automations host holds one token per workspace and reports each one's state

**Status:** resolved

**Blocked by:** 14, 15

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/typescript.md`, ADRs 0046, 0074, 0095, 0182, the spec (Automations hosts).

## What to build

- `nexul automations 1password <host> --workspace <slug>` (stdin, `--remove`), in the host unit's env,
  kept by `upgrade`, reusing ticket 15's plumbing.
- The server's host target gains the automation's workspace (`AutomationTarget` carries only id, name
  and token today).
- The supervisor reads the tokens, removes them from the environment the automation threads inherit,
  opens one client per token at start, and reports each workspace's state (`ready`, `token rejected`),
  stored on the host and shown in the web host list.

## Acceptance criteria

- [ ] A test asserts an automation thread's `process.env` holds no token.
- [ ] The host target carries the workspace over the hosts API.
- [ ] States reach the host list (component test, screenshots at 768, 1024 and 1440px).

Verify: `go test ./internal/automations/... ./internal/install/...`, `bun run typecheck` and
`bun run test` in `automations/` and `web/`.

## Comments

Built: `nexul automations 1password <host> --workspace <slug>` (one `NEXUL_OP_TOKEN_<SLUG>` variable per workspace in the host unit, kept by `upgrade`), the workspace slug on each host assignment, `POST /api/automation-hosts/self/managers` with migration 0149, and the per-workspace states in the web host list. The host takes every token out of `process.env` before any automation thread starts. A creation failure that is not the network reads as `token rejected`, as on the runner and the server. Checked against a real server with two bogus tokens, which reached the host list as rejected. The host does not retry an unreachable manager at start; a later pool resolve opens the client again.
