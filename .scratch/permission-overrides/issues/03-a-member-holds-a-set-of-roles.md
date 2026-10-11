# 03 — A member holds a set of roles (still one each)

**Status:** done

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

- [x] An empty `access.Matrix` diff on a production copy, in the PR. Replaced by a fixture: no production copy was
      read. A schema-0097 database built from the list parity fixture plus a second workspace with its own Owner, a
      member of both, workspace allows and denies, play exclusions, and rules for roles on docs, plays and a project
      gives the same `TestAccessMatrix_OfACopy` output on master and on this branch, 222 rows, byte for byte.
- [x] `TestRoleSet_EveryEvaluatorAgrees`: a member given two roles in a test gets the same union from
      `WorkspacePermissions`, `requireAction`, the invitation SQL and the owner list.
- [x] `TestStatements_LiveFramesCostOneReadPerPersonOnceWarm` keeps its count; a member's roles cost 2 statements or
      fewer, joined: one statement now, pinned by `TestStatements_RoleSetCostsOneReadAtAnySize`.
- [x] The tenancy and invitation suites stay green with only their fixtures changed; `/me`'s expected body gains
      `roles`.
- [x] The tool count is unchanged.

## Notes for later slices

- `workspace_list` carries the caller's set as `your_roles`, because its `roles` already lists the workspace's roles
  when an `id` is given. Slice 11 decides whether that name stays.
- `roles.requireAction` now answers through `access.WorkspacePermissions`, so a Team allow or deny of `roles:write` or
  `roles:clone` counts there as it already did everywhere else.
