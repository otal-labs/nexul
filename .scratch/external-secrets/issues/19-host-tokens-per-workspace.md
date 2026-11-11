# 19 — An automations host holds one token per workspace and reports each one's state

**Status:** ready-for-agent

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
