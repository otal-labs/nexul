# 15 — Shell jobs on a computer

**Status:** ready-for-agent

**Blocked by:** 06

Read first: `practices/go.md`, `practices/architecture.md`, `practices/mcp.md` (sections 3, 4, 6 to 8),
`practices/react-guide.md`, `practices/design-language.md`, `practices/testing.md`, ADRs 0091, 0138, the
spec (Later: shell jobs, Access and privacy).

## What to build

- The runner's job: `shell_run`/`shell_cancel`/`shell_output`/`shell_result`; bash, else sh, on Linux and
  macOS, PowerShell on Windows; no stdin, `NEXUL_*` stripped, timeout 60 s default and 1 h at most, 4 at a
  time, output at up to 64 KB a second with skipped-line markers. Refused unless the unit's env carries
  the install-time opt-in (`nexul install computer --allow-shell`).
- `runners.shell_enabled` (default off), flipped by the owner on the computer row; turning it off cancels
  running jobs.
- `runner_shell_jobs` with its 45-day purge (the audit log's loop pattern), output capped at 1 MiB.
- HTTP: start, cancel, list and get on the owner's computer; a per-viewer live socket for a running job.
  MCP: `command_run` (destructive, open-world, 5-minute cap, last 64 KB of output and the job id).
- The computer row's Commands section: the switch, the history with output, a running job live.

## Acceptance criteria

- [ ] `TestShellJobs_RefusedWithoutTheComputersOwnOptIn` (server switch on, unit opt-in off: refused by the
      runner, recorded as refused).
- [ ] `TestShellJobs_OnlyTheOwnerStartsOrReadsThem`: a workspace Owner and an every-bit role get 404 over
      HTTP and MCP, and see no job in any list.
- [ ] `TestShellJobs_NeverReachAnotherUsersSocket` and the job topics are members-only.
- [ ] An audit row is written per start, holding no command text.
- [ ] A disconnect fails the running job; a timeout kills the process tree.
- [ ] The tool count stays inside `toolBudget`, or the PR carries the ADR the budget rule asks for.
