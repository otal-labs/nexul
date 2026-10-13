# 18 — Share a computer: Run commands

**Status:** ready-for-agent

**Blocked by:** 15. The computer-rule prerequisite has merged.

Read first: `practices/go.md` (section 17), `practices/architecture.md` (sections 2, 3, 6),
`practices/react-guide.md`, `practices/mcp.md`, ADRs 0097, 0102, 0140, 0146, 0148, the spec (Later: sharing a
computer, Access and privacy), ADR 0148 (Computers).

Sharing is built on the shipped computer rules (ADR 0148): their storage, who may write a rule, the
checks, the immediate revoke and the privacy tests are the existing prerequisite. This ticket is the
runner-side half for "Run commands". There is no `computer_grants` table.

## What to build

- Shell jobs honour the computer rule's Run commands permission: the caller's start, list and read go through
  the existing computer check, and still need both of the computer's switches on. A grantee sees
  "<owner>'s <computer> (shared)" with name and online state only (ADR 0148), and may start shell jobs on it.
- The owner's sharing dialog on the computer row, writing the computer rule through
  `permission_overwrite_update` (named people only), with the warning the spec requires: the grantee's commands
  run as the owner's own OS user on the owner's computer. Reuse the shipped permission row
  component with two states.
- The grant follows the grantee's identity: their own web session, their plays and `@Agent` turns, and
  `command_run` with their own token. Nothing else gets in through it.
- Revoking, through the rule change, drops the grantee's queued jobs on that computer as "access revoked" and
  cancels their running ones, killing the process and recording "cancelled: access revoked".
- The owner's run log of what grantees ran (who, when, what, how long, how it ended, shell output); no agent
  transcript. A grantee sees only their own jobs.
- MCP: sharing is `permission_overwrite_update` on the computer (ADR 0148); `computer_list` shows the rules
  to the owner. No new tool.

## Acceptance criteria

- [ ] `TestShellJobs_GranteeNeedsRunCommands`: a rule with See only is refused, a rule with Run commands starts a
      job under both switches, and either switch off refuses it.
- [ ] `TestShellJobs_FollowTheGranteesIdentity`: the grantee's own token starts a job; a third person's token,
      and the owner's workspace Owner without a rule naming them, get 404.
- [ ] `TestShellJobs_RevokeDropsQueuedAndCancelsRunningJobs`, with no restart or cache wait.
- [ ] `TestGrantee_SeesNoFactsOrOtherPeoplesJobs`, and the owner's run log carries no transcript.
- [ ] The dialog shows the OS-user warning, and only the computer's owner sees it.
