# 09 — Personal runner on Windows

**Status:** ready-for-agent

**Blocked by:** 03, 07, 08

Read first: `practices/go.md`, `practices/testing.md`, ADR 0073, `internal/install/servicehost_windows.go`,
`website/public/runner.ps1`, `website/public/tunnel.ps1`, the spec (Installing a personal runner).

## What to build

`nexul install computer` on Windows, after macOS (owner, 2026-10-10): into `%LOCALAPPDATA%\Nexul`, running
under the installing person's own account as a service or the closest equivalent, the T3 Code desktop app
through winget when missing (ticket 07), minting with `~/.t3/bin/t3.cmd`. `computer.ps1` wrapper. Self-update
(ADR 0052's start-and-exit) and `nexul uninstall computer` work under it.

Open technical detail, decided in this ticket: a Windows service under a named account needs that person's
password or the log-on-as-service right, and the installer does not prompt. If neither is workable, the
closest equivalent is the per-user Scheduled Task at log on with restart on failure, which needs no
administrator and matches the desktop app only running while the person is logged in. Record the choice and why
in the PR; the spec's Windows row follows it.

## Acceptance criteria

- [ ] Unit tests for the service or task definition and paths.
- [ ] A run on a GitHub Windows runner: install, connect, relay a descriptor request, uninstall; reported in
      the PR.
- [ ] The dialog's Windows tab is live.
