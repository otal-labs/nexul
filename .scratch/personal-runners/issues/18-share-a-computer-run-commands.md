# 18 — Share a computer: Run commands

**Status:** ready-for-agent

**Blocked by:** 15

Read first: `practices/go.md` (section 17), `practices/architecture.md` (sections 2, 3, 6),
`practices/react-guide.md`, `practices/mcp.md`, ADRs 0097, 0102, 0140, the spec (Later: sharing a computer,
Access and privacy).

## What to build

- `computer_grants` as the spec defines it, with "Run agents" stored but not yet honoured (ticket 19).
- The owner grants and revokes per person from the computer row, with the warning the spec requires in
  the dialog. Grantees see "<owner>'s <computer> (shared)" with name and online state only, and may start
  shell jobs on it under the computer's two switches.
- The check order from the spec, in one query per check. A consumer of membership and account events
  deletes grants that no longer share a workspace or whose either account is disabled or removed.
- The grant follows the grantee's identity: their own web session, their plays and `@Agent` turns, and
  `command_run` with their own token. Nothing else gets in through it.
- Revoking refuses the next start, drops the grantee's queued jobs on that computer as "access revoked",
  and cancels their running ones, killing the process and recording "cancelled: access revoked".
- The owner's run log of what grantees ran (who, when, what, how long, how it ended, shell output); no
  agent transcript. A grantee sees only their own jobs.
- `computer.grant_changed` (outbox, members-only) reaching the owner and the grantee.
- MCP: grants are read on `computer_list` and changed through an existing computer tool's patch fields if
  one fits; a new tool needs the reason `practices/mcp.md` section 4 asks for.

## Acceptance criteria

- [ ] `TestGrants_OnlyTheOwnerGrants`, `TestGrants_NobodyGrantsThemselves`,
      `TestGrants_AdminCannotGrantOnAnotherPersonsComputer`, `TestGrants_EndWhenNoWorkspaceIsShared`,
      `TestGrants_EndWhenEitherAccountIsDisabled`.
- [ ] `TestGrants_GranteeCannotReshareOrRaiseALevel`.
- [ ] `TestGrants_FollowTheGranteesIdentity`: the grantee's own token starts a job; a third person's token,
      and the owner's workspace Owner, get 404.
- [ ] `TestGrants_RevokeDropsQueuedAndCancelsRunningJobs`, with no restart or cache wait.
- [ ] `TestGrantee_SeesNoFactsOrOtherPeoplesJobs`, and the owner's run log carries no transcript.
