# 15 — Shell jobs on a computer

**Status:** ready-for-agent

**Blocked by:** 06, permission-overrides 01

Read first: `practices/go.md`, `practices/architecture.md`, `practices/mcp.md` (sections 3, 4, 6 to 8),
`practices/react-guide.md`, `practices/design-language.md`, `practices/testing.md`, ADRs 0087, 0088, 0091, 0138,
`internal/platform/permissions`, the spec (Later: shell jobs, Access and privacy).

Build after any permissions rework the owner starts (this ticket adds a permission).

## What to build

- The runner's job: `shell_run`/`shell_cancel`/`shell_output`/`shell_result`; bash, else sh, on Linux and
  macOS, PowerShell on Windows; no stdin, `NEXUL_*` stripped, timeout 60 s default and 1 h at most, 4 at a
  time, output at up to 64 KB a second with skipped-line markers. A computer's runner installs with shell
  jobs on; `nexul install computer --no-shell` and `nexul shell on|off`, run at the computer, set the unit's
  own opt-in, and the runner refuses `shell_run` when it is off.
- `runners.shell_enabled`, on by default for a personal runner (its owner is the person who installed it, not
  the instance's owner), with a per-computer off switch on the computer row; turning it off cancels running
  jobs. Sharing stays off until the owner grants it (ticket 18).
- `runner_shell_jobs`: the command line, who ran it, when and the exit code kept forever; the output, capped at
  1 MiB, deleted after 30 days by a daily purge loop (the audit log's loop pattern) that empties the column
  and keeps the row. The job record is exempt from the audit log's 45-day purge.
- Read computer activity: the permission `computer_activity:read` (instance area, in
  `internal/platform/permissions`; the Owner role has it by default, no other role on upgrade, with the
  migration that grants it to existing Owner roles). Audit rows for actions on a computer (a shell start, and
  the computer routes: add, rename, remove, pair) name the computer and its owner and, for a shell start, the
  job, whose command line is read from the job record. They are returned only to holders of the permission:
  `audit:read` alone returns none of them, filtered in the query. The audit log gains a computer activity view
  for holders, showing computer, owner, who ran it, command, exit code and, while kept, the output.
- Each computer's page tells its owner: "Commands run here are recorded and can be read by people with Read
  computer activity." Facts stay owner-only; the permission does not reach them.
- HTTP: start, cancel, list and get on the owner's computer; a per-viewer live socket for a running job.
  MCP: `command_run` (destructive, open-world, 5-minute cap, last 64 KB of output and the job id).
- The computer row's Commands section: the switch, the history with output, a running job live.

## Acceptance criteria

- [ ] `TestShellJobs_RefusedWithoutTheComputersOwnOptIn` (server switch on, unit opt-in off: refused by the
      runner, recorded as refused), and `TestShellJobs_OnByDefaultForThePersonWhoInstalledIt` (not the
      instance's owner, unless they installed it).
- [ ] `TestShellJobs_OnlyTheOwnerStartsOrReadsThem`: a workspace Owner and an every-bit role get 404 over
      HTTP and MCP, and see no job in any list.
- [ ] `TestShellJobs_NeverReachAnotherUsersSocket` and the job topics are members-only.
- [ ] An audit row is written per start, naming the computer and the job; `TestComputerActivity_AuditReadAloneSeesNone`
      (a caller with `audit:read` and every other bit but `computer_activity:read` gets no computer rows), and
      `TestComputerActivity_OwnerRoleReadsCommandsAndOutput` (default Owner role, command, exit code, output).
- [ ] `TestComputerActivity_DoesNotShowFactsOrComputers`: the permission lists no computer, shows no facts,
      and a computer id still answers 404.
- [ ] `TestShellJobs_OutputPurgedAfter30DaysKeepsTheRecord`: the row's command, requester, time and exit code
      stay at 31 days and at a year, the output is empty, and the 45-day audit purge does not remove the job.
- [ ] The computer page shows the sentence about Read computer activity verbatim.
- [ ] A disconnect fails the running job; a timeout kills the process tree.
- [ ] The tool count stays inside `toolBudget`, or the PR carries the ADR the budget rule asks for.
