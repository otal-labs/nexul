# What the permission rework touches in the code

Research for ticket 02. Source: `origin/master` at `27742a10`. The model is the one decided in ticket 01, and
this file maps where it lands in the code without re-deciding it. Paths are repo-relative. A `:N` or `:N-M` is
a line or range in that file at that commit.

Production facts were read on 2026-10-11 from a read-only connection (`mode=ro`). Production is at migration
`0088`, behind master's `0095`.

| Fact | Value |
|---|---|
| `permission_overwrites` rows | 32 `doc` and 2 `project`. No `workspace` rows, no `play` rows. |
| The 32 doc rows | Every one is the creator's own row (`user_id = docs.created_by`), and every one is exactly `CreatorGrant` with an empty deny. No doc has an empty `created_by`. |
| The 2 project rows | One is a dormant row of an unrestricted member (all 27 project-area levels). The other is the Restricted client's `tickets:read`, `tickets:write`. |
| Memberships | 20 in all, 1 of them Restricted. That person is a Client in the workspace with 3 projects. |
| Channels | 8 text and 5 voice channels, every one public. There are no private channels. The Restricted client's workspace has 3 of them. |
| Open invitations | 1, carrying no `allow_json`, `deny_json`, `restricted` or `project_access_json`. None of the 10 grants ever made carries them. `workspace_invites` is empty. |
| Plays | 56 (8 seeded in each of 7 workspaces). 7 auto plays, all off. 312 trails. |
| Role bits | Clients hold no `plays:*` and no `projects:read`. Developer holds `plays:read` without `plays:run`. Senior Developer and Administrator hold both. |

So on production the wipe deletes 2 rows and one `restricted = 1`, and keeps all 32 doc rows. The code paths below
still have to handle every shape, because other installs upgrade too.

---

## 1. Check paths and list filters

Every access answer goes through the access service. Two more evaluators live outside it (1.9).

### 1.1 The evaluator core (`internal/access/usecase.go`)

| Function | Where | Today | What changes |
|---|---|---|---|
| `decide` | `:95-110` | Owner, then hidden, then Member, then role and workspace layers, then the resource's own row | Takes several roles' union. The "hidden" step becomes "not in this project" for every member, not only a Restricted one. Plays gain a per-play See step. Returns a source (ADR 0148). |
| `workspaceLayers` struct | `:114-122` | `role permissions.Set` (one role), `overwrite` (the Team row), `restricted`, `project` | `role` becomes the set of roles (ids plus the union). `overwrite` goes with the wipe unless a workspace person rule survives (Q4). `restricted` goes. `project` becomes the project rules for the person and their roles. |
| `workspaceLayers()` | `:124-141` | `memberRole` (one role), then the workspace overwrite | Reads the member's role set. With no Team rows there is nothing more to read. |
| `layers()` | `:145-155` | Reads the project row only `if ws.restricted` | Reads project rules (person plus each held role) for **every** member. This is the main new read on the hot path. |
| `hidden()` | `:158-160` | `restricted && inProject && len(project)==0` | Becomes "the project's rules do not list the person or any role they hold". Applies to every non-Owner. |
| `has()` | `:162-176` | A Restricted member's instance area is false, and their project area comes from `project` alone | Project-area actions inside a project come from the role union plus the project's rules. The rules for project-area actions with **no project named** need a decision (Q3). |
| `WorkspacePermissions` | `:191-196` | Feeds `/api/workspaces/{id}/me` and every `requireWorkspacePermission` | For project areas it means what Q3 decides. |
| `ProjectPermissions` | `:200-213` | `opens=false` when hidden | Unchanged in shape, but now every member can be hidden. |
| `CanInProject` | `:216-225` | | Unchanged shape. |
| `applyOverwrite` | `:228-236` | Deny beats allow in one row | "Allow wins between role rules", then the person rule beats the roles (ticket 01). This needs a merge over several role rows before the person row. |
| `Can`, `CanDocs` | `:240-265` | One read of the person's doc rows for the lot | Must also read the doc rules of the person's roles, in the same statement (the ADR 0148 performance plan). |
| `DocsWith` | `:270-295` | `ListByUser(doc, user)`: person rows only | Adds role-targeted doc rows. Must stay bounded by the number of rules, not docs. Doc lists now call it on the **non-search** path too (1.4). |
| `ListUsers` | `:447-469` | Open to `accounts:read`, or to someone holding a **person** row that allows `permissions:write` on a doc (`HasAllowAny` `:456`) | Must count role rules too. ADR 0148 already flags this. |
| `canManage` | `:487-493` | Doc: `permissions:write` on the doc. Play: `plays:write` in the workspace | Gains project (the new "Manage this project's access"), and channel and folder later. |

### 1.2 The gate (`internal/access/gate.go`)

| Function | Where | What changes |
|---|---|---|
| `require` | `:27-47` | `ws.hidden()` → not found (`:33-35`) now applies to every member. `action == ""` means membership only (`:37-39`). This interacts with zero roles (Q2). |
| `RequireProject` | `:51-60` | Unchanged shape. Every member can now get not found. |
| `RequireAnywhere`, `HoldsAnywhere`, `PermissionsAnywhere`, `workspacesOf` | `:64-77`, `:146-157`, `:161-179`, `:181-192` | `workspacesOf` reads `UnrestrictedWorkspaceIDsForUser` (`:186`). With Restricted gone it becomes `WorkspaceIDsForUser`, so a Client role carrying an instance bit would start to count. On production no Client role carries one. |
| `ProjectsAnywhere` | `:81-101` | The loop over workspaces is unchanged. |
| `CallerProjects` | `:105-123` | Unchanged shape. The answer for a role-only member goes from "every project" to "the listed ones". |
| `ProjectsWith` | `:269-290` | Batches the project rows only for a Restricted member (`:278-282`). It must batch project rules (person plus roles) for **every** member, in one statement per workspace. Otherwise `s.check` per project (`:284-288`) reads a row per project, and the `TestStatements_*` guards fail (section 9). |
| `IsOwner`, `IsRestricted` | `:242-251` | `IsRestricted` goes. Its one caller is chat (1.6). `IsOwner` reads "holds the Owner role among their roles". |

### 1.3 The memo (`internal/access/memo.go`, ADR 0135)

- Keys `:138-145`. `overwriteKey{resourceType, resourceID, userID}` is keyed by person. Role rules need a key with
  no user, such as `{resourceType, resourceID, roleID}`, or one statement that reads the person's and roles' rows
  together, keyed by `{resourceType, resourceID, userID}` once the role set is known.
- `memberRole` `:147-151` remembers one `RoleInfo`. It becomes the role set.
- `overwrites` (batched) `:159-164` → `repo.GetMany` (`internal/access/repo.go:11-26`). It needs a role-aware
  variant.
- The commit fence is unchanged. A role-rule write commits through the serializer like any write.

### 1.4 Lists that filter by access (ADR 0140)

| List | Where | Today | What changes |
|---|---|---|---|
| Docs, paged | `internal/docs/usecase.go:259-273`, `listScope` `:277-301` | Without a query: `CallerProjects(Member)` only, and unreadable docs come back with `can_open=false` (`listItems` `:314-337`). With a query: `DocsWith` | "Unreadable means not found": the no-query path must also filter by `DocsWith(DocsRead)`, and `can_open` stops meaning anything in a list. |
| Docs, whole | `List` `:236-242`, `ListByProject` `:245-254`, `toListItems` `:303-312` | Title disclosed with `can_open=false` | Same change. The comment at `:302` and ADR 0028's sentence "the doc list do show an inert title" change too. |
| Projects | `internal/workspace/usecase.go:144-166` | `permissions.Filter` by `RequireProject(Member)`, one check per row | Rides the new `layers()`. The list is small, but it should use the `ProjectsWith` batch. |
| Tickets, deploys, notifications, memories, queued runs | `CallerProjects`/`ProjectsAnywhere` callers (ADR 0140) | | No code change beyond the gate. The answers move. |
| People who may open a project | `internal/tenancy/project_access.go:197-216` | `ProjectPermissions` per person, N people | Under opt-in this is "who the project's rules list". It should become one SQL question, and is a performance risk as written. |
| Team project access | `teamProjects` `:220-243` | Reads `ListTeamProjectAccess`, one visibility check per project | Replaced by the project rules view (section 7). |

### 1.5 Plays (detail in section 8)

`ListApplicable` (`internal/plays/usecase.go:143-172`) already checks `plays:run` per play through the play row
(`:166`). `List` (`:125-138`) and `Get` (`:175-180`) check only the workspace's `plays:read`. A per-play See has to
reach both, plus the trails and auto-play paths.

### 1.6 Channels (they decide whether Restricted can be deleted at all)

- `readsChannel` (`internal/chat/usecase.go:145-154`): a public channel is read by every member **except a
  Restricted member**. `refuseRestricted` (`:230-240`, called at `:217` and `:324`) stops a Restricted member
  creating or switching to a public channel. The `Standing` seam is `:55-62`, wired at
  `server/cmd/services.go:295`.
- So today the Restricted flag also hides **every public channel** (ADR 0098: "a Restricted member's channels are
  exactly the private ones they are in"). Deleting Restricted without channel rules in the same release lets the
  production client read the 3 public channels of her workspace, which she cannot read today (Q1).
- The web reads `role?.restricted` to force new conversations private
  (`web/src/hooks/useNewConversationDialogs.tsx:13`, `web/src/components/chat/ChannelSettingsCard.tsx:29`).

### 1.7 Unreadable means not found

- Trails: `GetTrail` hides a trail on a project the reader cannot open (`internal/plays/run_trail.go:24-26`), but
  `requireTrailAccess` returns **forbidden** for a missing `plays:read` (`:110-119`).
- Plays: `require` (`internal/plays/usecase.go:455-460`) returns forbidden. A play hidden by See must be not found
  instead.
- Docs: `require(DocsRead)` on `Get` (`internal/docs/usecase.go:229`) returns forbidden for a member who can open
  the project but not the doc. The decided model wants not found.

### 1.8 The doc mention pill

- The server already resolves an unreadable doc to its title with `can_open=false`: `resolveDoc`
  (`internal/mentions/usecase.go:315-326`), using `Can(DocsRead)` (`:371-380`). The web already draws the lock
  (`web/src/components/doc/mention/MentionChip.tsx:52`), inert and unlinked (`:70-80`). What is missing is the "You
  don't have access" tooltip.
- `resolveDoc` reads the doc by id with no workspace or project check. Under opt-in, the title of a doc in a
  project the reader is not in still shows, while the project and everything in it is not found. Ticket pills do
  the same: `resolveTicket` (`:275-293`) discloses title, status, type and developer, with `can_open` from
  `RequireProject(TicketsRead)` (`:364-369`) (Q5).
- ADR 0028 keeps search filtered and lets the chip disclose. That stands.

### 1.9 Evaluators outside the access service

| Where | What it evaluates | Several roles? |
|---|---|---|
| `internal/roles/usecase.go:303-322` (`requireAction`) | Looks up the actor's **one** role and checks `actorRole.Permissions.Has(action)`. It ignores Team overrides entirely. It gates role create, update and delete (`:298-301`). | Must union roles, and should call `access.WorkspacePermissions` instead. |
| `internal/platform/storage/invitations_repo.go:644-677` (`creatorAccess`) | The role set plus the workspace overwrite, Restricted narrowed to workspace areas, in SQL through `GetInvitationCreatorWorkspaceAccess` (`queries/invitations.sql:86-94`) | Must read the role set, and loses the overwrite and Restricted branches. |
| `queries/invitations.sql:30-62` (`ListManageableInvitationHeaders`) | `members:write` evaluated with `instr` over one `m.role_id`'s permissions and the workspace overwrite | Must be rewritten over the role set. |
| `queries/auth.sql:39-42` (`ListActiveOwnerIDs`) | Owner = `JOIN roles r ON r.id = m.role_id AND r.is_owner_role = 1` | Over the role set. It decides `/api/auth/me`'s owner wizard (`internal/auth/service.go:512-532`). |

---

## 2. One role per membership

### 2.1 Schema

- `workspace_members.role_id TEXT NOT NULL REFERENCES roles(id)`, with no `ON DELETE`
  (`internal/platform/storage/migrations/0001_schema.sql:317-323`). Zero roles needs it gone or nullable, which is a
  table rebuild in SQLite. Several roles needs a join table, for example `member_roles(user_id, workspace_id,
  role_id)`.
- `workspace_members.restricted` (`migrations/0051_project_access.sql:2`).
- `roles.is_owner_role` with `idx_roles_owner_singleton` (`0001_schema.sql:307-316`). There is one Owner role per
  workspace. Under several roles, Owner is one role in a member's set (Q-list item 7).
- `invitation_grants.role_id` and the PK `(invitation_id, workspace_id)` (`migrations/0002_private_invitations.sql:13-20`)
  allow one role per workspace per invitation.
- `workspace_invites.role_id` (`0001_schema.sql:333-341`). This is the old login-invite table (0 rows on
  production), still written by `InviteMember`.
- Deleting a role a member holds fails today on the foreign key (`repo.Delete`, `internal/roles/usecase.go:174-189`,
  has no check of its own). With a join table cascading on role delete, deleting a role can leave a member with
  zero roles, which is now allowed.

### 2.2 sqlc queries that read or write the single role

- `queries/tenancy.sql`: `AddWorkspaceMember` `:20-21`, `GetWorkspaceMemberRole` `:23-24`,
  `SetWorkspaceMemberRole` `:63-64`, `ListTeamMemberships` `:85-95` (joins `m.role_id`), the `workspace_invites`
  queries `:66-83`.
- `queries/invitations.sql`: `CreateInvitationGrant` `:5-7`, `ListManageableInvitationHeaders` `:30-62`,
  `GetInvitationCreatorWorkspaceAccess` `:86-94`, `GetInvitationMember` `:96-99`, `AddInvitationMember` `:101-104`.
- `queries/auth.sql:39-42` (`ListActiveOwnerIDs`).
- Membership read as `RoleIDFor`, a role lookup used as a membership test:
  - `internal/platform/storage/tenancy_repo.go:153-159`
  - `internal/tenancy/usecase.go:198`, `:248`, `:621`
  - `internal/tenancy/people.go:45`, `:105`
  - `server/cmd/wire_gates.go:412-425` (chat's `IsMember`)
  - `internal/roles/usecase.go:130-140`

  These keep working only if membership stays a row in `workspace_members`.

### 2.3 Tenancy use-cases (`internal/tenancy/usecase.go`)

| Use-case | Where | Assumes one role |
|---|---|---|
| `Create` (workspace creator gets Owner) | `:54-95` (`:84`) | Writes `RoleID` |
| `BindDefaultWorkspaceOwner` | `:239-266` | Idempotence check through `RoleIDFor` |
| `MemberRoleID`, `MemberRoleName` | `:269-305` | One id, one name. They feed `/api/workspaces/{id}/me` and MCP `workspace_list` |
| `requireHoldsRole` | `:332-338` | One role's set |
| `InviteMember` (login invite) | `:341-385` | One `roleID`. It still writes `workspace_invites` |
| `ListWorkspaceMembers` | `:401-427` | `MemberView.RoleID` (`internal/tenancy/model.go:115-119`) |
| `RemoveMember` | `:430-451` | Refuses the Owner by checking one role |
| `ChangeMemberRole` | `:454-493` | Replaces the one role. The hold check covers only the new role (`:482`) |
| `AddMember` | `:591-633` | Requires a `roleID` (`:598-600`), so zero roles is impossible today |
| `SetMemberOverrides` | `:636-683` | The Team allow and deny. Wiped (section 3) |
| `ResolvePendingInvites` | `:717-732` | One role per pending invite |

`Member`, `TeamMembership` and `InvitationGrant` carry `RoleID` and `RoleName` (`internal/tenancy/model.go:33-39`,
`:83-93`, `:165-177`).

### 2.4 How roles are given, and the hold-what-you-give check

The rule (ADR 0088): nobody grants a bit they do not hold. It is implemented as `permissions.RequireHeld`
(`internal/platform/permissions/gate.go:49-51`) at these call sites:

1. `tenancy.requireHolds` (`internal/tenancy/usecase.go:327-329`), which is `RequireHeld(grant,
   WorkspacePermissions(actor))`. Used by `InviteMember` (`:361`), `ChangeMemberRole` (`:482`), `AddMember`
   (`:611`), `SetMemberOverrides` (`:671-674`, counting a lifted deny as a grant), and `requireHoldsEveryProject`
   (`internal/tenancy/project_access.go:55-65`).
2. `SetProjectAccess` against what the giver holds **on that project** (`project_access.go:69-102`, `:94`).
3. Invitations: `validateInvitationPackage` → `invitationGrantActions` (role plus allow,
   `invitations_repo.go:535-557`) checked against `creatorAccess` (`:417-462`), and `requireCreatorHoldsProjects`
   per project (`:466-487`). It runs again on redeem (`:311`) and on list (`:181`).
4. Roles: `roles.requireHolds` (`internal/roles/usecase.go:289-295`) for create, update and clone.
5. **Missing** on doc sharing and play exclusions. `setGrants` (`internal/access/usecase.go:392-416`) checks only
   `canManage`, so a holder of `permissions:write` on a doc can grant `docs:delete` without holding it. ADR 0148
   promises the check on every rule.

Under several roles, "may give role R" is still `RequireHeld(R.permissions, held)`, with `held` as the union over
the giver's roles. Removing a role is never a grant. Giving several roles at once is the union of their sets.

### 2.5 HTTP surfaces

- `GET /api/workspaces/{id}/me` (`internal/tenancy/handler.go:191-214`) returns `role_name` (one), `permissions`,
  `restricted` and `projects` (Restricted only). Under opt-in **every** member needs per-project actions, so
  `projects` becomes unconditional, `restricted` goes and `role_name` becomes a list.
- `PATCH /api/workspaces/{id}/members/{userID}` (`handler.go:227-238`, `patchMember` `:311-335`) carries
  `role_id`, `allow`, `deny`, `every_project` and `project_access`.
- `PUT …/members/{userID}` and `POST …/members` (`:216-223`, `:251-295`) take one `role_id`.
- `GET /api/auth/me` (`internal/auth/service.go:512-532`, `internal/auth/model.go:206-212`): `instance_permissions`
  comes from `PermissionsAnywhere` (Restricted no longer narrows it). `needs_owner_wizard` comes from
  `ListActiveOwnerIDs`.
- The invitation acceptance page shows `role_id` and `role_name` per grant (`internal/auth/model.go:81-88`,
  `server/cmd/wire_invitations.go:67-72`, `web/src/components/invitation/InvitationGrantSummary.tsx:14`).

### 2.6 Web

| What | Where |
|---|---|
| Role picker, one value | `web/src/components/team/TeamRoleSelect.tsx`, used in `TeamWorkspacePanel.tsx:39-46` and `TeamAddWorkspacePopover.tsx:61` |
| Team row summary "role in workspace" | `web/src/components/team/TeamPersonRow.tsx:19` |
| Draft model with `roleId` | `web/src/models/MemberDraft.tsx:7-11`, `:44`, `:52-61`, `:72`, `:85`, `:109` |
| Team model `role_id`, `role_name` | `web/src/models/Team.tsx:43-50`. `roleHolders` matches one role (`:92-97`) |
| Invitation model and form, one role | `web/src/models/Invitation.tsx:18-23`, `:57-68`. `web/src/components/member/InvitationGrantRow.tsx:27-35`, `:58`. `InvitationRow.tsx:21` |
| Account menu shows the role name | `web/src/components/AccountMenuTrigger.tsx:54` |

### 2.7 MCP (detail in section 6)

`account_list` and `account_update`, `invitation_create` and `invitation_list`, and `workspace_list`'s `role`
(`internal/mcp/composite/workspaces.go:51`, `:202-233`).

### 2.8 The phone app and client-core

- `client-core/permissions.ts:32-58`: `MyWorkspaceInfo` has `role_name`, `restricted?` and `projects?`.
  `workspaceWidePermissions` and `projectPermissions` return the flat `permissions` for anyone not Restricted.
  Under opt-in, every member's project answer is per project, so both functions change for web and phone together.
- Phone: `native/src/hooks/WorkspaceHooks.tsx:31-52` (the `getMyRole` query, refreshed on `role.updated` at
  `:38`), `native/src/push/usePushNotificationRouting.tsx:28`, `native/src/components/AreaGate.tsx`.
- The phone docs list shows locked docs (`native/src/components/docs/DocRow.tsx:22-35`,
  `DocsListScreen.tsx:28`). These leave the list once the server filters (1.4).
- Live: the phone refetches everything only when a frame's `user_id` is the viewer
  (`native/src/hooks/useLiveEvents.tsx:13-26`). A role-targeted rule names no user (section 5).

### 2.9 Live audience

`memberFrame` reaches the person and every member (`server/cmd/live_audience.go:239-248`). `role.updated` reaches
every member (`workspaceFrame` `:251-256`, mapped at `:84`). Several roles changes nothing here. The payloads do:
`MemberEvent` (`internal/tenancy/events.go:54-60`) names no role, which is fine.

---

## 3. Every writer and reader of `permission_overwrites`, Restricted, Every project and Project access

### 3.1 Writers

| Writer | Where | Resource type | Shape |
|---|---|---|---|
| `GrantCreator` (on doc create) | `internal/access/usecase.go:358-366`, called from `internal/docs/usecase.go:120-131` (`:127`) outside the doc's own transaction | `doc` | allow `CreatorGrant` (`internal/platform/permissions/permissions.go:286-287`). **Must survive the wipe.** |
| Doc sharing (`SetGrants` → `applyGrant`) | `usecase.go:377-379`, `:497-524` | `doc` | Add or remove allow. Clears a deny on grant |
| Play exclusion (`SetPlayGrants`) | `usecase.go:383-390`, `:507-513` | `play` | Add or remove deny of `plays:run` |
| Team override (`SetMemberOverrides` → `SetOverrides`) | `internal/tenancy/usecase.go:636-683`, `internal/platform/storage/tenancy_repo.go:315-334` | `workspace` | allow and deny |
| Project access (`SetProjectAccess`) | `internal/tenancy/project_access.go:69-102`, `tenancy_repo.go:249-267` | `project` | allow only |
| Invitation redeem | `invitations_repo.go:345-367`. `AddInvitationProjectAccess` (`queries/invitations.sql:112-115`), `AddInvitationOverwrite` (`:117-120`) | `project`, `workspace` | Insert, do nothing on conflict |
| Account delete | `queries/auth.sql:48-49` | all | Delete by user |
| Member removed | `tenancy_repo.go:174-189` (`:184`), `queries/tenancy.sql:41-44` | `project` only | The workspace override and doc rows stay |
| Doc deleted | `DeleteByDoc` `usecase.go:369-374`, from `internal/docs/usecase.go:505` | `doc` | Delete by resource |
| Project deleted | `internal/platform/storage/workspace_repo.go:191` | `project` | Delete by resource |
| Play deleted | none | `play` | Rows are orphaned (there are none on production) |
| Migrations | `0057_docs_lock_permission.sql:11-27`, `0082_auto_plays.sql:37-58` | all | Rewrote allow and deny sets |

### 3.2 Readers

| Reader | Where |
|---|---|
| The evaluator | `internal/access/memo.go:153-164` → `access_repo.go:26-75` (`Get`, `GetMany`, `ListByUser`), `queries/access.sql:1-28` |
| Doc and play grant lists | `listGrants` `usecase.go:428-444` → `ListOverwritesByResource` (`access.sql:4-5`) |
| User picker gate | `HasAllowAny` → `CountOverwriteAllowAny` (`access.sql:19-21`, an `instr` over JSON) |
| Team | `ListTeamMemberships` (`tenancy.sql:85-95`), `ListMemberProjectAccess` (`:46-51`), `ListTeamProjectAccess` (`:53-58`) |
| Team overrides read before a change | `tenancy_repo.go:300-313` |
| Project settings "People with access" and the delete impact | `ListProjectRestrictedAccess` (`queries/projects.sql:46-53`). `internal/workspace/usecase.go:170-183`, `:310-334` (`:326`). `GET /api/projects/{id}/access` (`internal/workspace/handler.go:120`, `:262-269`). `DeleteImpact.RestrictedMembers` (`internal/workspace/model.go:236-255`) |
| Invitations | `GetInvitationCreatorWorkspaceAccess`, `ListManageableInvitationHeaders`, `GetInvitationCreatorProjectAccess` (`invitations.sql:30-62`, `:86-94`, `:109-110`) |
| Migration 0038 backfill of `docs.created_by` | `migrations/0038_doc_created_by.sql`. The earliest doc row became the creator |

### 3.3 Every use of Restricted, Every project and Project access

- Go:
  - `internal/access/model.go:28-33`
  - `usecase.go:114-176`
  - `gate.go:146-192`, `:248-251`, `:278-282`
  - `server/cmd/wire_access.go:37-47` (`Restricted: m.Restricted`), `:106-112`
  - `internal/tenancy/model.go:32-62`, `:90-98`, `:163-177`
  - `internal/tenancy/project_access.go` (the whole file)
  - `internal/tenancy/handler.go:189-196`, `:225-238`, `:324-333`
  - `internal/tenancy/invitations_service.go:39`, `:166`
  - `internal/tenancy/invitations_mcp.go:18-30`, `:104-112`
  - `internal/platform/storage/tenancy_repo.go:128-132`, `:199-283`, `:354`, `:366`
  - `invitations_repo.go:466-487`, `:505-531`, `:578-619`, `:634-684`, `:723-755`
  - `internal/chat/usecase.go:55-62`, `:145-154`, `:230-240`
  - `internal/memories/interview.go:161` (a path kept for a Restricted reader)
  - `internal/workspace/*` (`ListRestrictedAccess`, `RestrictedMembers`)
  - `internal/mcp/composite/accounts.go` (section 6)
  - `internal/roles/mcp.go:60`
- Web:
  - `web/src/components/access/EveryProjectRow.tsx`
  - `ProjectAccessBlock.tsx`, `ProjectAccessRow.tsx`
  - `web/src/components/team/TeamProjectAccess.tsx`
  - `web/src/components/member/InvitationProjectAccess.tsx`
  - `web/src/components/settings/ProjectPeopleAccessSection.tsx`, `RestrictedMembersLoseAccess.tsx`,
    `ProjectDangerZoneSection.tsx:44-49`
  - `web/src/models/Team.tsx:14-50`, `MemberDraft.tsx`, `Invitation.tsx:22-23`, `:60-70`, `Project.tsx:124`
  - `web/src/hooks/AccessHooks.tsx:12-27`, `MemoryHooks.tsx:173`, `useNewConversationDialogs.tsx:13`
  - `web/src/components/chat/ChannelSettingsCard.tsx:29`
- Docs: `CONTEXT.md:734-770` (the permission entry, Restricted member `:749`, Project access `:758`, Every project
  `:765`), Invitation `:781` and Team `:793`. ADRs 0042, 0085,
  0087, 0088, 0097, 0098, 0102, 0135, 0140. Practices: `practices/native.md:77`,
  `practices/design-language.md:756`, `:797`. Guide pages under `website/src/content/docs/docs/guide/`:
  `people-and-access.md`, `projects-and-repositories.md`, `plays.md`, `api-and-tokens.md`, `paired-computers.md`.

### 3.4 The doc creator's grant through the wipe

- Identified by `docs.created_by` (`migrations/0038_doc_created_by.sql`), set on create
  (`internal/docs/usecase.go:110`) and on clone (`:169`). On production it is set for all 32 docs, each with
  exactly one overwrite row, the creator's.
- Under opt-in, a creator's doc rule matters only inside a project they are listed in, because a project they are
  not in is not found before the chain walks inward. A creator left out of the grid loses their own docs until
  listed.
- If the table is rebuilt with a `target` column (ADR 0148), the kept rows become `target = 'person'`.

---

## 4. The wipe migration

The migration is forward-only, the next free number on master (the last is `0095`; note there are two `0092_*`
files). It runs once on every install that upgrades.

### 4.1 Rows deleted from `permission_overwrites`

| Delete | Predicate | Production |
|---|---|---|
| Team overrides | `resource_type = 'workspace'` | 0 |
| Project access, active and dormant | `resource_type = 'project'` | 2 |
| Play exclusions, and any allow rows (a test writes one: `server/cmd/live_audience_test.go:125`) | `resource_type = 'play'` | 0 |
| Doc sharing with anyone but the creator | `resource_type = 'doc' AND user_id IS NOT (SELECT created_by FROM docs WHERE id = resource_id)` | 0 |
| Doc rows of docs that no longer exist | `resource_type = 'doc' AND resource_id NOT IN (SELECT id FROM docs)` | 0 |

### 4.2 Rows kept

The creator's own doc row: `resource_type = 'doc' AND user_id = docs.created_by` (32 on production, all equal to
`CreatorGrant`). Options for its contents:

- **(a)** Keep it exactly as stored. This is the smallest change and loses nothing a creator set on themselves.
- **(b)** Reset it to `CreatorGrant` with an empty deny. This is a cleaner start, and on production it is
  identical to (a).

### 4.3 Columns and tables

| Column or table | Action |
|---|---|
| `workspace_members.restricted` | Drop it (`ALTER TABLE … DROP COLUMN`, SQLite 3.35+), or zero it and leave it dead. Production has 1 set. |
| `workspace_members.role_id` | Copy each row's role into the new role-set table ("who holds which role" is kept). Then drop the column, or rebuild the table without `NOT NULL` so zero roles can exist. |
| `invitation_grants.allow_json`, `deny_json`, `restricted`, `project_access_json` | See 4.4. |
| `invitation_grants.role_id` | Becomes a role set (a JSON list or a child table). Copy the one role across. |
| `workspace_invites` (login invites, empty on production) | Becomes a role set, or is dropped with `InviteMember` if the link invitations replace it. |
| A new per-workspace marker | Drives the Owner's one-time "Who's in each project" grid. It is set by the migration for every existing workspace, or only those with projects (one production workspace has none). |

### 4.4 Invitations that are still open

These are unredeemed and unexpired. Their per-person parts have no place left to land.

| Option | What happens | Trade-off |
|---|---|---|
| **A. Strip** | Clear `allow_json` and `deny_json` to `[]`, set `restricted = 0` and `project_access_json = '[]'`, and keep the role. The person joins with the role and lands in no project, exactly like an existing member after the wipe. | The link keeps working, and the inviter may not notice the projects they picked are gone. |
| **B. Delete the ones carrying any of them** | Delete the invitation rows (grants cascade). Invitations with only a role survive. | Honest, but the inviter must remake them. A migration writes no outbox event, so no `invitation.deleted` frame goes out. |
| **C. Delete every open invitation** | Simplest. | It also drops plain role-only links. |
| **D. Translate on redeem** | Turn `project_access_json` into the new project person rules when the link is used. | Contradicts the wipe: someone invited before the upgrade would keep project access that members lost. |

On production every option is the same: 1 open invitation, carrying none of the four. Whatever is chosen,
`validateInvitationProjects` and `requireCreatorHoldsProjects` (`invitations_repo.go:466-487`, `:580-619`) go, and
re-validation on redeem and list (`:181`, `:311`) must stop reading `restricted`.

### 4.5 What the code must do at the same time (or the release is broken)

- With no project listing anyone, only Owners see projects. Everything acting as a non-Owner loses its projects
  until the grid is filled:
  - personal access tokens and automations created by a member (ADR 0087: a token acts as its creator)
  - auto plays running "on" a developer. `whyNobody` (`internal/plays/autoplay_match.go:364-376`) records "may not
    run here". The 7 auto plays on production are off.
  - T3 project links, which are hidden for a project you cannot open (ADR 0102)
  - notifications (`internal/workspace/notification_usecase.go:205-216`)
  - play runs started before the upgrade, which reattach (ADR 0119)
- A project's creator must be written into the project's rules at creation. `workspace.Service.create`
  (`internal/workspace/usecase.go:74-114`) writes nothing today, so the creator's own new project would be not
  found to them.

---

## 5. Live push and audience rules

| Topic | Where | Today | What changes |
|---|---|---|---|
| `access.grant.changed` | Payload `internal/access/events.go:12-17` (`enum:"doc,play,project"` at `:13`). Tenancy shares the topic (`internal/tenancy/events.go:15-16`, `ProjectAccessEvent` `:37-44`). Audience `grantFrame` (`server/cmd/live_audience.go:227-237`): the named person, plus `members:write` holders for a project. Automation scope `grantScope` (`server/cmd/automation_scope.go:106`, `:360-…`) | One person per frame | The payload gains `target` and `role_id` **beside** `user_id`, never in its place (Hit every surface). The enum gains `channel` and `folder` when they ship. The audience becomes the role's holders for a role rule, and the project's managers become "Manage this project's access" holders. Then run `make event-schemas` and `make live-topics`, and the contract test `internal/eventcatalog/contract_test.go` keeps the contract only growing. |
| `workspace.member.updated` | `internal/tenancy/usecase.go:704-714` | Carries every project id of the workspace, because a role, override or Every project change could move any of them | Unchanged. It also covers a role added or removed. |
| `role.updated` | `internal/roles/events.go:5-16`, audience `workspaceFrame` (`live_audience.go:84`, `:251-256`) | | Unchanged. A role's set now reaches its holders in every project that lists the role. |
| Play frames | `playFrame` `live_audience.go:444-465` | `plays:read` in the workspace, or `plays:run` on that play | Becomes per-play See. |
| Play run frames | `playRunFrame` `:427-442` | Follows the target | Add per-play See, or the starter. |
| Auto play frames | `autoPlayFrame` `:467-480` | `autoplays:read` | Add per-play See of the auto play's play. |
| Installation frames | `installationFrame` `:265-275` | `Require(workspace, projects:read)` with no project named | Depends on Q3. |
| Chat frames on a public channel | through `readsChannel` | Hidden from a Restricted member | Become channel rules. |

Clients:

- The web refetches everything on `access.grant.changed` only when `user_id` is the viewer
  (`web/src/hooks/useLiveEvents.tsx:51-56`). It already handles a frame that names no holder for `role.updated`
  (`:57-62`): it refetches `/me` and compares. A role-targeted grant frame needs the same path.
- `projectFollower` (`web/src/hooks/ProjectHooks.tsx:217-223`) refreshes the "People with access" key on project
  grants. It changes with that card.
- `TeamHooks.tsx:81-82` follows both topics.
- The phone dispatches an access change only on its own `user_id` (`native/src/hooks/useLiveEvents.tsx:13-26`), and
  refreshes only `getMyRole` on `role.updated` (`native/src/hooks/WorkspaceHooks.tsx:38`).
- The request-count guards move: `web/src/hooks/liveRequests.test.tsx:329` (member updated), `:381`
  (`role.updated` → `["/api/auth/me"]`), and `native/src/hooks/requestCounts.test.tsx`.

---

## 6. MCP

The surface holds **110 tools**, counted from `registryTools` on this commit. The guard is `assert.Less(len,
toolBudget)` with `toolBudget = 111` (`internal/mcp/surface_test.go:21`, `:36-38`), so the effective ceiling is
110. **There is no headroom at all.** Any new capability has to ride an existing tool, as ticket 01 already says.

| Tool | Where | Fields that change |
|---|---|---|
| `permission_overwrite_list` | `internal/access/mcp.go:18-22`, `:41-77` | `resource_type` is "doc or play" (`:19`, `:108-110`). It gains `project`, then `channel` and `folder`. Each row says its target (`target`, `user_id` or `role_id`). The description describes play exclusion (`:43-46`). |
| `permission_overwrite_update` | `:24-30`, `:79-106` | It gains `target`, `role_ids` and `state`. `grant` on a play means "lift the exclusion" (`:29`), which inverts once a play carries See rules. The description (`:81-84`) changes. This is how project membership ("People in this project") and channel hiding are written by agents. |
| `account_update` | `internal/mcp/composite/accounts.go:71-90`, `:124-150`, `:171-190` | `role_id` (one) → several roles (add or remove, or a replacing list). `allow` and `deny` are deprecated with the wipe (Q4). `every_project` and `project_access` go, or `project_access` becomes person rules on the project. The description at `:126-137` changes. |
| `account_list` | `:54-65`, `:97-122`, `:247-248` | `role_id` and `role_name` → a list. `allow`, `deny`, `every_project` and `projects` go. The description `:99-109`. |
| `invitation_create` and `invitation_list` | `internal/tenancy/invitations_mcp.go:18-30`, `:45-77`, `:93-115` | `role_id` → several. `allow`, `deny`, `every_project` and `project_access` go (option A or B in 4.4). |
| `workspace_list` | `internal/mcp/composite/workspaces.go:51`, `:202-233` | `role` (one name) → a list. |
| `role_update` description | `internal/roles/mcp.go:60` | Mentions `every_project` and `project_access`. |
| `play_list`, `play_run`, `trail_list`, `trail_update` | section 8 | Filter by per-play See, and not found for a play the caller cannot see. |
| `project_create` | `internal/mcp/composite/projects.go` | The creator is always in. Whether it takes "who's in it" roles is open (it would ride this tool, not a new one). |

`account_update`'s `every_project` and `project_access` are documented in practices and the guide
(`api-and-tokens.md`), so removing them is a published-contract change. The MCP practice's deprecation rules apply.

---

## 7. Web editors and surfaces

### 7.1 Components the panel replaces

| Component | Where it is used | Replaced by |
|---|---|---|
| `PermissionsForm` (doc sharing and play exclusion) | `web/src/components/access/PermissionsForm.tsx`. Opened from `web/src/pages/DocPage.tsx:36-40` and `web/src/components/play/PlayRow.tsx:31-43` (title "Exclude users", count of `deny` holders `:34-35`) | The permissions panel (ticket 03) |
| `TeamOverridesForm` (Allow levels and Deny grid) | `web/src/components/team/TeamOverridesForm.tsx`, from `TeamWorkspacePanel.tsx:50-58` | Gone with the wipe (Q4), or the panel on the workspace |
| `TeamProjectAccess` → `ProjectAccessBlock` → `EveryProjectRow`, `ProjectAccessRow` | `web/src/components/team/TeamProjectAccess.tsx`, `TeamWorkspacePanel.tsx:61`. `web/src/components/access/{ProjectAccessBlock,EveryProjectRow,ProjectAccessRow}.tsx` | A view of the person's project rules, or gone |
| `InvitationProjectAccess` | `web/src/components/member/InvitationProjectAccess.tsx`, `InvitationGrantRow.tsx:70` | Gone (an invite gives roles only) |
| `ProjectPeopleAccessSection` ("People with access", read-only, Restricted members only) | `web/src/components/settings/ProjectPeopleAccessSection.tsx`, from `ProjectSettingsContent.tsx:27` (gated on `canManageMembers`) | "People in this project" (a list of roles and people), opening the panel |
| `RestrictedMembersLoseAccess` | `web/src/components/settings/RestrictedMembersLoseAccess.tsx`, `ProjectDangerZoneSection.tsx:44-49` | Gone, or "N people lose this project" from the project's rules |
| `TeamRoleSelect` (one role) | `TeamWorkspacePanel.tsx:39-46`, `TeamAddWorkspacePopover.tsx:61` | A multi-role control (ticket 04) |
| Shared pieces | `PermissionGrid.tsx`, `PermissionLevels.tsx`, `PermissionDomainRow.tsx`, `PermissionLevelControl.tsx`, `RoleLevelSections.tsx` (the role editor's Workspace and Every project split) | Reused, or the three-state control. The role editor's "Every project" heading changes meaning. |

### 7.2 Where a right-click menu attaches

There is no context-menu primitive in `web/src/components/ui/`, only `dropdown-menu.tsx`. The `radix-ui`
umbrella package (`web/package.json:50`) includes ContextMenu, so the shadcn `context-menu` item installs through
the CLI without a new dependency. Confirm at install.

| Thing | Row | Existing ⋯ or menu |
|---|---|---|
| Doc | `web/src/components/doc/DocListRow.tsx:17` | `DocActionsMenu.tsx:40-41` already has "Permissions" |
| Doc folder | `web/src/components/doc/DocFolderHeader.tsx:18` | none |
| Project | `web/src/components/sidebar/ProjectSection.tsx:20`, `ProjectSwitcherMenu.tsx:13` | Project settings page |
| Channel | `web/src/components/sidebar/ChannelSidebarRow.tsx:15`, `VoiceChannelSidebarRow.tsx:20` | `ChannelSettingsDialog`/`ChannelSettingsCard` |
| Play | `web/src/components/play/PlayRow.tsx:28` | The row's "Exclude users" action |

### 7.3 The doc mention pill

`web/src/components/doc/mention/MentionChip.tsx:39-82` already renders the title, a lock (`:52`) and an inert span
(`:70-80`) when `can_open` is false. It needs only the "You don't have access" tooltip, plus a decision on ticket
pills (Q5). The phone renders doc bodies through its own path, which should be checked at build time.

### 7.4 Play lists, run menus, auto plays and trails that must respect See

| Surface | Where | Reads |
|---|---|---|
| Plays settings list | `web/src/components/settings/PlaySettingsSection.tsx:46` → `useFetchWorkspacePlays` (`web/src/hooks/PlayHooks.tsx:20`) | `GET /api/workspaces/{id}/plays` → `plays.List` |
| Auto plays per play | `PlayHooks.tsx:28-111`, `AutoPlaySignals.tsx`, `AutoPlaysPausedRow.tsx` | `ListAutoPlays`, `AutoPlaysByPlay` |
| Run menus and buttons | `PlaysMenu.tsx:28`, `PlaysRailSection.tsx:18`, `PlayButton.tsx`, `RunItButton.tsx`, `PlayRunDialog.tsx`. Docs: `useDocBuiltinPlays` (`PlayHooks.tsx:127-135`), `ClarificationActions.tsx:14`, `DocDetail.tsx`. Interview: `InterviewRunButton.tsx`, `InterviewAuditButton.tsx`, `InterviewDraftButton.tsx`, `InterviewMemoryColumn.tsx`. Tickets: `TicketPageBody.tsx` | `ListApplicable` (`/plays/applicable`) |
| Trails | `TrailSection.tsx:23`, `TrailHooks.tsx`, `TrailContinueForm.tsx` | `ListTrails`, `GetTrail`, `ActiveTrails` |
| Queued runs in a thread | `web/src/components/chat/QueueEventLine.tsx` | `GetQueue` (labels of plays the viewer may not see) |

### 7.5 The project wizard's "Who's in it" step

- Steps are listed twice: the server in `internal/workspace/model.go:48`
  (`SetupSteps = {"project","repository","service","env","reach","branches"}`), the web in
  `web/src/models/ProjectWizard.tsx:3-34` (with labels, descriptions and `setupResumePath`).
- Content: `web/src/components/wizard/ProjectWizardStepContent.tsx:26`, `web/src/pages/ProjectWizardPage.tsx`,
  `web/src/stores/projectWizardStore.tsx`, `web/src/hooks/useWizardSetup.tsx`, `useWizardNavigation.tsx`.
- Setup storage: `projects.setup_steps` JSON (`migrations/0085_project_setup.sql`). A new step id is additive.
  Projects made before it count as finished, and `setupResumePath` skips steps already marked.
- The project is created by `CreateInSetup` (`internal/workspace/usecase.go:69-71`) at the Info step. The creator
  must be listed in the same write (4.5), with the step pre-ticking the creator's own roles.

---

## 8. Plays: every place a play is listed or run

| Path | Where | Today's check | With per-play See |
|---|---|---|---|
| `plays.List` (settings, `play_list` without type) | `internal/plays/usecase.go:125-138`, `internal/plays/mcp.go:155-163` | workspace `plays:read` | Filter out unseen plays. A list is config-sized (ADR 0140 allows in-memory paging), so one batched rule read keeps it flat. |
| `plays.Get`, `BuiltinPlay` | `:175-180`, `:375-381` | workspace `plays:read`, forbidden | Not found for an unseen play. `BuiltinPlay` also feeds the template layers (`server/cmd/wire_templates.go:105-122`). |
| `ListApplicable` (run menus, `play_list` with type) | `:143-172` | `plays:run` on the play (`:166`), one `HasPermission` per play, memoised | Adds See. |
| `checkPlay` (every start: button, MCP `play_run`, the queue) | `internal/plays/run_launch.go:542-563` (`:559`) | Project membership plus `plays:run` on the play | Adds See, with not found rather than forbidden. |
| Auto play matching | `internal/plays/autoplay_match.go:326-…` (`matchOne`), `whyNobody` `:364-376` | `plays:run` on the play for the person it runs on | Adds See for that person. |
| Auto play lists | `internal/plays/autoplay_usecase.go:24-37`, `:41-63` | workspace `autoplays:read` | Only the auto plays of visible plays. |
| Queue for a target, queued runs of a play | `internal/plays/queue.go:80-…`, `:104-130` | `autoplays:read`, then projects | Leaves out unseen plays' items. `QueuedForPlay` gains a See check. |
| Trails | `internal/plays/run_trail.go:15-31`, `:40-57`, `:60-85` (`:80`), `:88-108`, `:110-119` | workspace `plays:read` to read **any** trail, plus `docs:thread` for a doc | Ticket 01 says "past runs stay visible to whoever started them and to people who can see the play". Starter-based reading is **new**: today a starter without `plays:read` cannot read their own trail. On production the Developer role has `plays:read` without `plays:run`. |
| Clarify gate (docs) | `server/cmd/wire_gates.go:327-350` (`clarifyPlayGate`) | `plays:run` on the Clarify play | Adds See. |
| Decisions check | `internal/plays/decisions_check.go:60` | Server-side `List` | Server call, unaffected. |
| Live | `live_audience.go:427-480` | section 5 | |
| Integrations scope | `internal/integrations/gateway.go:94-95` | `plays:run` token scope | Unchanged (scopes are out of scope). |
| Edit and exclusions | `Update` `usecase.go:216-239`, `unseenExclusions` `:445-453` (project exclusions, not people) | `plays:write` | Unchanged. `ExcludedProjectIDs` is a per-project switch, not a permission. |
| Web and phone | 7.4 | | The phone shows no plays (no `/plays` reads in `native/src`). |

What See is stored as is open (Q-list item 8). Today the catalog row is
`plays: read, write, delete, run` (`internal/platform/permissions/permissions.go:139`). `plays:read` means "list
and read definitions" for the whole workspace, and `plays:run` is the only per-play bit (`SetPlayGrants` refuses
any other, `internal/access/usecase.go:383-390`).

---

## 9. Performance guards that will likely move

`server/cmd/access_memo_test.go`:

| Guard | Where | Why it moves |
|---|---|---|
| `TestStatements_ListsCostTheSameAtAnyLength` | `:266-310` | It compares few rows against many, not absolute counts, so it holds if every new read is batched. The risks are these. **"docs list"**: `ListByProject` starts filtering by `DocsWith`, which adds role doc rules. **"chat conversation list" and "chat unread counts"**: the Restricted standing per public channel becomes a channel rule read per channel unless batched. **"a play's queued runs"**: `QueuedForPlay` gains a See check and project rules. **All paths**: the fixture's `uReader` must be listed in `pGeneral` by a project rule. Otherwise opt-in makes the paths answer not found or empty, and the guard passes for the wrong reason. |
| `TestStatements_TicketSearchFetchesOnlyEachResult` | `:312-332` | Holds while the project-rule read is one per project, memoised. |
| `TestStatements_LiveFramesCostOneReadPerPersonOnceWarm` | `:334-359` | Public-channel frames read `IsRestricted` today. The channel-rule read must sit under the same memo fence. Each `MemberRole` is 2 statements today (`server/cmd/wire_access.go:37-47`: membership, then role). A role set should stay at 2 or fewer, joined. |
| `TestIntegration_PermissionSuitesAnswerTheSameWithAMemo`, `TestIntegration_LiveAudienceMemo_FollowsEveryChange` | `:65-150` (`uClient` Restricted at `:136-140`) | The fixture is Restricted-based. |
| `TestProjectsWith_ReadsARestrictedMembersAccessOnce` | `internal/access/memo_test.go:64` | Becomes "reads every member's project rules once". |
| `TestCanDocs_ReadsEachLayerOnceForTheLot` | `internal/access/restricted_test.go:184` | Counts the Restricted layers. |

`server/cmd/list_paging_test.go`:

| Guard | Where | Why it moves |
|---|---|---|
| `TestRows_MessageListReadsOnlyItsPage`, `TestRows_ListToolsReadOnlyTheirPage` | `:63-85`, `:87-131` | They run as `uOwner`, whose bypass is unchanged, so the row counts should hold. But they do not exercise a non-Owner's `DocsWith` on the no-query doc path. A non-Owner variant would guard the new filter. |
| `TestListTools_ShowWhatTheRowByRowCheckShowed` | `:221-…`, fixture `listParityFixture` `:172-219` | `restrictedClient` (`:178`) and person doc rows (`:191-193`). `doc_list can_open` parity (`openedOnFirstPage`) changes meaning: listed equals openable once unreadable docs leave lists. |

Paths with no guard today that become hot:

- `ListProjectPeople` (`internal/tenancy/project_access.go:197-216`): a check per person.
- The `workspace.Service.List` per-row filter (`internal/workspace/usecase.go:156`).

The spec's numbers (ticket 02 and ticket 04 of the original spec) should include `doc_list`, `ticket_list` and a
live frame with project rules on every project.

---

## 10. Tests to delete or rewrite

**Delete. They encode Restricted, Every project, Project access or the Team override:**

- `internal/access/restricted_test.go`: `TestRestrictedMember_*` (`:34`, `:125`, `:138`). Keep `TestCanDocs_*`,
  rewritten.
- `internal/tenancy/project_access_test.go`: `TestSetProjectAccess`, `TestSetEveryProject`,
  `TestMemberProjectsAndTeam`, `TestHandler_ProjectAccess` (`:72-256`). `TestListProjectPeople` (`:211`) is
  rewritten.
- `internal/platform/storage/project_access_test.go`: all four (`:17`, `:64`, `:128`, `:147`). The migration 0051
  test stays only as long as 0051 runs before the wipe.
- `server/cmd/project_access_integration_test.go`: `TestIntegration_RestrictedMember*` (`:55`, `:143`, `:177`,
  `:197`). The notice, trail and live cases are rewritten as "not in the project".
- `internal/chat/private_test.go:364` `TestRestrictedMember_ReadsDMsTheirPrivateChannelsAndReadableThreads`
  (becomes a channel-rule test).
- `internal/memories/interview_test.go:225` (a Restricted reader).
- `internal/tenancy/usecase_test.go:1266` `TestSetMemberOverrides`, if the workspace person rule goes (Q4).
- `internal/mcp/composite/accounts_test.go:157` (`project_access` ordering).
- `internal/access/memo_test.go:64`.
- Web:
  - `web/src/components/settings/ProjectPeopleAccessSection.test.tsx`
  - the Every project cases in `TeamPersonDialog.test.tsx`, `TeamSection.test.tsx`, `CreateInvitationForm.test.tsx`,
    `web/src/models/Invitation.test.tsx`
  - `ProjectDangerZoneSection.test.tsx` (restricted members)
  - `ChannelSettings.test.tsx` (restricted private-only)
  - `RoleSettingsSection.test.tsx` (the area split)
  - `PermissionsForm.test.tsx` (replaced with the panel)
- `client-core/permissions.test.ts` (Restricted branches).
- Phone: `native/src/components/AreaGate.test.tsx`, `native/src/components/board/BoardScreen.test.tsx` (Restricted
  cases).

**Rewrite. They assume one role or person-only rows:**

- `internal/access/usecase_test.go`: `TestHasPermission_Precedence` (`:300`), `TestWorkspacePermissions` (`:446`),
  `TestSetGrants` and `TestListGrants` (`:565`, `:637`), `TestSetPlayGrants` and `TestListPlayGrants` (`:679`,
  `:728`, exclusion semantics), `TestListUsers` (`:758`).
- `internal/access/gate_test.go`: `TestRequire`, `TestPermissionsAnywhere`, `TestWorkspacesWith`.
- `internal/access/memo_test.go`: `TestProjectsWith_*`, `TestCallerProjects_*`, `TestDocsWith_*`.
- `internal/access/handler_test.go` and `mcp_test.go` (`TestGrantTools_PlayExclusion` `:132`).
- `internal/platform/storage/access_integration_test.go`: `TestIntegration_ListDisclosure` (`:183-185`, which
  pins the title-with-lock in lists that the model removes) and `TestIntegration_HasPermission_WorkspacePrecedence`
  (`:332`).
- `internal/platform/storage/invitations_repo_test.go`: `TestInvitationsRepo_Redeem_ExistingMembershipPreservesRoleAndOverwrite`
  (`:195`) and every grant-shape case.
- `internal/tenancy/invitations_service_test.go:137` (`Preview_HidesPermissionOverwrites`) and
  `invitations_mcp_test.go`.
- `internal/tenancy/usecase_test.go`: the role-change and add-member cases (one `role_id`).
- `internal/roles/usecase_test.go` and `mcp_test.go` (`requireAction` on one role).
- `server/cmd/permissions_integration_test.go`: the fixture `seedPermFixture` (`:195-…`) gives each person one role
  and no project rule. Under opt-in every role-holder sees no project, so `TestIntegration_PermissionTable`
  (`:249`) and `TestIntegration_GrantsNeverExceedTheGiver` (`:545`) need project rules in the fixture.
- `server/cmd/live_audience_test.go:117` (`PlayFramesReachWhoSeesThePlay`, which sets a play **allow** row `:125`)
  and `:149`.
- `server/cmd/access_memo_test.go` and `list_paging_test.go` (section 9), `mentions_integration_test.go`,
  `clarification_integration_test.go`.
- Migration tests `migration_0037_test.go`, `0057`, `0082`, `0086`: they seed `workspace_members.role_id` and
  `permission_overwrites`. They keep running against their own schema step, but break if a later migration's
  fixture helper assumes the new shape.
- Web and phone: `liveRequests.test.tsx`, `native/src/hooks/useLiveEvents.test.tsx:46`, `:136-152`,
  `requestCounts.test.tsx`.

**Replaced as a gate.** The original spec's matrix gate and `TestMigration_EffectiveAccessUnchanged` are dropped
(ticket 01). The wipe needs its own migration test instead: a fixture with every row kind at the previous schema,
upgraded, asserting which rows survive (creator rows) and that `restricted` and the per-person rows are gone.

---

## Questions this raises for the owner

1. **Channels in the wipe release.** Restricted also hides every public channel today (`internal/chat/usecase.go:145-154`).
   Production's one Restricted client reads none of her workspace's 3 channels today, all public, and would read
   all of them after the wipe. Do channel rules (at least "deny a role or person `channels:read` on a channel")
   ship in the wipe release, or is that widening accepted until channels ship?
2. **Zero roles "sees nothing" against membership-only reads.** ADR 0087 makes People, public channels, `#general`,
   DMs and the inbox readable with membership alone (`gate.go:37-39`). A zero-role member still sees those. Is
   "nothing" meant to cover them, which turns membership-only reads into role reads, or only projects?
3. **What "in a project" is, and what a project-area bit means with no project named.** `projects:read` already
   gates repositories and the delete impact (`internal/workspace/usecase.go:304`, `:462`) and installation frames.
   The draft `d0580b18` made "In this project" a permission outside the role grid. Separately: creating a project
   needs `projects:write` with no project named. So does a role's project-area bit count outside any project
   (today's From role), or never (today's Restricted)?
4. **Does a workspace-wide person rule survive?** The wipe deletes Team's allow and deny. Do Team, `account_update`
   `allow`/`deny` and `SetMemberOverrides` stay for new rules, or go with "roles only"?
5. **Pills across the not-found boundary.** `resolveDoc` and `resolveTicket` disclose titles by id, ticket status
   and developer included, with no project check (`internal/mentions/usecase.go:275-326`). Under opt-in that
   discloses content of a project that is otherwise not found. Ticket 01 decided docs only. What about ticket
   pills?
6. **Open invitations:** strip, delete those carrying per-person parts, delete all, or translate (4.4)?
7. **Owner among several roles.** Can the Owner also hold other roles? The use-case guards "never given, changed or
   removed" (`internal/tenancy/usecase.go:454-493`) become set rules.
8. **Play See.** Is it a new verb, or `plays:read` applied per play? What does a role grant by default? Is the
   starter's own trail readable without `plays:read`, which is new?
9. **Between upgrade and grid,** every non-Owner token, automation, auto play, T3 link and notification loses its
   projects (4.5). Is a notice to non-Owners ("your Owner is setting up project access") wanted?
10. **The hold-what-you-give check is missing on doc and play grants** (`internal/access/usecase.go:392-416`).
    Fix it in the rework, as ADR 0148 implies?
