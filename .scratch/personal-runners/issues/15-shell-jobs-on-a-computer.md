# 15 — Shell jobs on a computer: Read computer activity

**Status:** ready-for-agent

**Blocked by:** nothing. Shell jobs themselves shipped (ADR 0209): the runner's job, both switches,
`runner_shell_jobs` with its 30-day output purge, the HTTP routes and live socket, `command_run`, and the
computer row's Commands section. What remains is who besides the owner may read that record.

Read first: `practices/go.md`, `practices/architecture.md`, `practices/react-guide.md`,
`practices/design-language.md`, `practices/testing.md`, ADRs 0087, 0088, 0138, 0148, 0209,
`internal/platform/permissions`, the spec (Access and privacy, rule 5).

## What to build

- The permission `computer_activity:read` (instance area, in `internal/platform/permissions`; the Owner role has it
  by default, no other role on upgrade, with the migration that grants it to existing Owner roles).
- Audit rows for actions on a computer (a command's start, and the computer routes: add, rename, remove, pair)
  name the computer and its owner and, for a command, the job, whose command line is read from the job record.
  They are returned only to holders of the permission: `audit:read` alone returns none of them, filtered in the
  query. The audit log gains a computer activity view for holders, showing computer, owner, who ran it, command,
  exit code and, while kept, the output.
- Each computer's page tells its owner: "Commands run here are recorded and can be read by people with Read
  computer activity." Facts stay owner-only; the permission does not reach them.

## Acceptance criteria

- [ ] An audit row is written per start, naming the computer and the job; `TestComputerActivity_AuditReadAloneSeesNone`
      (a caller with `audit:read` and every other bit but `computer_activity:read` gets no computer rows), and
      `TestComputerActivity_OwnerRoleReadsCommandsAndOutput` (default Owner role, command, exit code, output).
- [ ] `TestComputerActivity_DoesNotShowFactsOrComputers`: the permission lists no computer, shows no facts,
      and a computer id still answers 404.
- [ ] The computer page shows the sentence about Read computer activity verbatim.
