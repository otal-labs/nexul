# 21 — Computer sharing

**Status:** done

**Blocked by:** 17

**Branch:** master, a normal PR, after the switch. It doesn't depend on the switch's model; it comes after it so the
wipe never touches a computer row.

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3, 6, 8), `practices/mcp.md`
(sections 4, 6, 7), `practices/testing.md` (sections 3, 8, 10), ADRs 0102, 0135, 0146, 0148, the spec (Computers),
`.scratch/personal-runners/spec.md` (Access and privacy, Later: sharing a computer).

A person's computer is the one entity the Owner's bypass does not reach. This slice builds the rules that let its owner
share it by name. The runner-side work is personal runners tickets 18 and 19, which build on this one.

## What to build

- Computer rules are `permission_overwrites` rows with `resource_type = 'computer'`, `target = 'person'` only. A role
  row can't be written on a computer. The rows are deleted with the computer and with either account, and a consumer
  of membership and account events deletes a rule whose person no longer shares a workspace with the owner.
- A computer permission set outside the role grid: **See this computer**, **Run agents**, **Run commands**. No role,
  invitation, integration install or token scope can hold them. Allowing Run agents or Run commands also allows See.
- A computer check, separate from the workspace chain and never memoised: signed in and active; the owner holds
  everything; otherwise the computer loads only through a rule naming the caller that allows See, while the owner is
  active and the two share a workspace; the caller holds exactly what it allows. No role, Owner or every-bit role
  reaches it. No match is not found.
- Only the owner writes rules, in Allow or Fallback (Fallback removes the person); Deny is refused; a grantee can't
  write, re-share or raise one. Revoking is immediate.
- `permission_overwrite_list` and `_update` accept `resource_type: computer`, not found to anyone else; explain covers
  computers (the owner for any named person, a grantee for themselves).
- Every list of a person's computers (`computer_list`, the Computers page, the run dialog, project links) is one
  query: their own plus those shared with them by name, the shared ones as "<owner>'s <computer> (shared)" with name
  and online state only.
- `computer.grant_changed` (outbox, members-only, ids and the permissions, never the computer's name), reaching only
  the owner and the person named. `make event-schemas`, `make live-topics`.
- `CONTEXT.md` (Computer sharing) and the guide page `paired-computers.md`.

## Acceptance criteria

- [x] `TestComputerRules_OwnerOrEveryBitRoleWithoutARuleGetsNotFound`, over HTTP and MCP, and the computer is in none
      of their lists, events or sockets.
- [x] `TestComputerRules_ANamedPersonHoldsExactlyWhatIsGranted`, including an Owner who is named.
- [x] `TestComputerRules_OnlyTheOwnerWrites`, `TestComputerRules_TargetsAreNamedPeopleOnly` (a role write and a Deny
      are refused), `TestComputerRules_RevokeIsImmediate`.
- [x] `TestComputerRules_EndWhenNoWorkspaceIsSharedOrAnAccountIsDisabled`, and a rule doesn't return on rejoining.
- [x] `TestComputerRules_EventsReachOnlyTheOwnerAndTheGrantee`; `computer_activity:read` alone still lists no
      computer.
- [x] The tool count is unchanged.
