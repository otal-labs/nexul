# 03 — A member holds a set of roles (still one each)

**Status:** ready-for-agent

**Blocked by:** 02

**Branch:** master, a normal PR, before the switch. Every writer still gives exactly one role. May take two sessions
on one branch.

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 1, 8), `practices/mcp.md`
(sections 4, 7, 11), `practices/typescript.md` (section 11), `practices/testing.md` (sections 3, 8, 10), ADRs 0024,
0061, 0082, 0085, 0088, 0148, the spec (Roles, Source and explain), the research (sections 1.9 and 2).

## What to build

- A migration adds `member_roles(user_id, workspace_id, role_id)`, cascading on role and account delete, backfills it
  from `workspace_members.role_id`, then rebuilds `workspace_members` without `role_id`. `restricted` stays until the
  switch.
- These read the set, as a union: the 13 sqlc queries and the tenancy use-cases (research 2.2, 2.3); `memberRole` and
  the memo; `roles.requireAction`, through `access.WorkspacePermissions`; the invitation SQL (`creatorAccess`,
  `ListManageableInvitationHeaders`); `ListActiveOwnerIDs`; `IsOwner` ("holds the Owner role among their roles").
- Membership stays a row in `workspace_members`, so `RoleIDFor` callers that test membership keep working through a
  membership read.
- Every writer still gives exactly one role. Deleting a role someone holds is still refused, now in the use-case.
- `/api/workspaces/{id}/me`, `GET /api/team`, `account_list` and `workspace_list` gain a `roles` list beside the
  single fields.

## Acceptance criteria

- [ ] An empty `access.Matrix` diff on a production copy, in the PR.
- [ ] `TestRoleSet_EveryEvaluatorAgrees`: a member given two roles in a test gets the same union from
      `WorkspacePermissions`, `requireAction`, the invitation SQL and the owner list.
- [ ] `TestStatements_LiveFramesCostOneReadPerPersonOnceWarm` keeps its count; a member's roles cost 2 statements or
      fewer, joined.
- [ ] The tenancy and invitation suites stay green with only their fixtures changed.
- [ ] The tool count is unchanged.
