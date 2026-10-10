# 09 — Personal runner on Windows

**Status:** done

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

- [x] Unit tests for the service or task definition and paths.
- [x] A run on a GitHub Windows runner: install, connect, relay a descriptor request, uninstall; reported in
      the PR.
- [x] The dialog's Windows tab is live.

## Comments

Built as a per-user Task Scheduler task at log on (`InteractiveToken`, `LeastPrivilege`, no administrator), not a
service: a service under the person's account needs their password, which the install never asks for, and T3 Code's
desktop app only runs while they are logged on anyway (ADR 0146; ADR 0052 for the update path).

- The task runs `nexul service-host nexul-computer`, which keeps the runner in a kill-on-close job, starts it again
  5 s after a failure, ends with its clean exit, and sets `NEXUL_SUPERVISED` so a self-update exits for the host to
  start the new binary.
- Everything lives in `%LOCALAPPDATA%\Nexul` and is the person's, so removal needs no root cleanup: a removed runner
  starts `nexul uninstall computer --yes` outside its task, which ends and deletes the task and deletes the folder and
  its user Path entry once it exits.
- `computer.ps1` hands over to `install.ps1`, which never elevates for a computer. `hostcred.ComputerCommands` renders
  `commands.windows`; the Add a computer dialog shows it under Windows.
- `.github/workflows/computer-windows-smoke.yml` (start by hand) installs on `windows-latest` through `computer.ps1`,
  checks the task and the runner's account, kills the runner and sees the host start it again, waits for connected
  and paired, relays a providers request to T3 Code, then removes the computer and checks the task, the folder, its
  Path entry and the runner process are gone; it passed on `windows-latest`. T3 Code there is its command line, since a
  runner without a screen cannot open the desktop app, so the install and the runner also find the command line's
  `~\.local\bin\t3.cmd` on Windows.
