# 12 — Computer rules: sharing a computer by name

**Status:** ready-for-agent

**Blocked by:** 01, 02

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3, 6, 8),
`practices/mcp.md` (sections 4, 6, 7), `practices/testing.md` (sections 3, 8, 10), ADRs 0097, 0102, 0135, 0146,
0148, the spec (Evaluation order, Computers), `.scratch/personal-runners/spec.md` (Access and privacy, Later:
sharing a computer).

A person's computer is the one entity the Owner's bypass does not reach. This ticket builds the rules that let
its owner share it by name, and the privacy that keeps everyone else out. The runner-side work (shell jobs for a
grantee, a starter's own token for Run agents, the attribution line, the warning dialog) is personal runners
tickets 18 and 19, which build on this one.

## What to build

- Computer rules are `permission_overwrites` rows with `resource_type = 'computer'`, `resource_id` the computer
  id, `target = 'person'` only. No Everyone and no role row can be written on a computer. The rows are deleted
  with the computer and with either account, and a consumer of membership and account events deletes a rule
  whose person no longer shares a workspace with the owner.
- A computer permission set of its own, outside the role grid: **See this computer**, **Run agents**, **Run
  commands**. No role, invitation, integration install or token scope can hold them. Allowing Run agents or
  Run commands also allows See.
- A computer check, separate from the workspace chain and never memoised across requests: signed in and active;
  the computer's owner holds everything; otherwise the computer loads only through a rule that names the caller
  and allows See, while the owner is active and the caller shares a workspace with them; the caller then holds
  exactly what that rule allows. No other path reaches it: not a role, a workspace or instance Owner, a role
  holding every bit, or a workspace person rule. No match is not found, never forbidden.
- Only the computer's owner writes rules: Allow or Fallback (Fallback removes the person); Deny is refused;
  a grantee cannot write, re-share or raise a rule. Revoking is immediate.
- `permission_overwrite_list` and `_update` accept `resource_type: computer`, resolved through ownership and
  returning not found to anyone else; explain covers computers (the owner explains for any named person, a
  grantee only for themselves). Sharing needs no new tool.
- Every list that shows a person's computers (`computer_list`, the Computers page, the run dialog, project
  links) is one query: the caller's own, plus the ones shared with them by name, showing the shared ones as
  "<owner>'s <computer> (shared)" with name and online state only.
- The event is `computer.grant_changed` (outbox, members-only, ids and the permissions, never the computer's
  name), reaching exactly the owner and the person named. `access.grant.changed` is not used for computers,
  since its audience reaches workspace managers. `make event-schemas` and `make live-topics`.
- `CONTEXT.md` (Computer sharing, and the Owner's exception under Permission overwrite) and the guide page
  `paired-computers.md`.

## Acceptance criteria

- [ ] `TestComputerRules_OwnerOrEveryBitRoleWithoutARuleGetsNotFound`: a workspace Owner, the instance Owner and
      a role holding every bit get 404, never 403, over HTTP and MCP, and the computer is in none of their
      lists, events or sockets.
- [ ] `TestComputerRules_ANamedPersonHoldsExactlyWhatIsGranted`: See only, See and Run commands, and all three,
      each refused outside the granted set; this holds for an Owner who is named too.
- [ ] `TestComputerRules_OnlyTheOwnerWrites`: a grantee, an admin, an Owner and the instance Owner cannot write,
      change or remove a rule, and a grantee cannot grant themselves more.
- [ ] `TestComputerRules_TargetsAreNamedPeopleOnly`: an Everyone or role write, and a Deny, are refused.
- [ ] `TestComputerRules_RevokeIsImmediate`: the next check after the write is refused with no cache wait.
- [ ] `TestComputerRules_EndWhenNoWorkspaceIsSharedOrAnAccountIsDisabled`, and a rule does not return when the
      person rejoins.
- [ ] `TestComputerRules_EventsReachOnlyTheOwnerAndTheGrantee`, and `computer_activity:read` alone still lists
      no computer.
- [ ] No role, invitation or token scope accepts a computer permission.
- [ ] The matrix diff on a production copy is empty, and the tool count is unchanged.
