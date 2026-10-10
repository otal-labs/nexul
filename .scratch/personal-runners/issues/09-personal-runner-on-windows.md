# 09 — Personal runner on Windows

**Status:** ready-for-agent

**Blocked by:** 03, 07

Read first: `practices/go.md`, `practices/testing.md`, ADR 0073, `internal/install/servicehost_windows.go`,
`website/public/runner.ps1`, `website/public/tunnel.ps1`, the spec (Installing a personal runner).

## What to build

`nexul install computer` on Windows without administrator: into `%LOCALAPPDATA%\Nexul`, a per-user
Scheduled Task at log on with restart on failure, the T3 Code desktop app through winget when missing
(ticket 07), minting with `~/.t3/bin/t3.cmd`. `computer.ps1` wrapper. Self-update (ADR 0052's
start-and-exit) and `nexul uninstall computer` work under the task.

## Acceptance criteria

- [ ] Unit tests for the task definition and paths.
- [ ] A run on a GitHub Windows runner: install, connect, relay a descriptor request, uninstall; reported in
      the PR.
- [ ] The dialog's Windows tab is live.
