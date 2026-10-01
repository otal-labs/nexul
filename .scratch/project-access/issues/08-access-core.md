# 08 — Access core: Restricted members and Project access on the server

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** tickets 01, 02, 03, 05; spec sections "Server checks" to "What the client is told"; ADR 0097

## What to build

The backend of Restricted members and Project access, every adapter except
MCP (ticket 11) and the web (ticket 12).

1. **Migration 0049** (`internal/platform/storage/migrations/`):
   `workspace_members.restricted INTEGER NOT NULL DEFAULT 0`;
   `invitation_grants.restricted INTEGER NOT NULL DEFAULT 0` and
   `invitation_grants.project_access_json TEXT NOT NULL DEFAULT '[]'`.
   Queries in `internal/platform/storage/queries/`, then `make sqlc`.
2. **Areas.** Each row of `domainTable`
   (`internal/platform/permissions/permissions.go`) gains its area,
   `project`, `workspace`, or `instance`, per the spec's table; `Info` and
   `GET /api/permissions/catalog` serve `area`. The `memories:clone` label
   stays as ticket 09 leaves it.
3. **Resolver** (`internal/access/usecase.go`, `gate.go`): for a restricted
   membership after the Owner bypass, workspace-area actions answer as
   today; instance-area actions are refused; project-area actions answer
   from the `permission_overwrites` row (`resource_type = "project"`) alone
   and are refused with no project; `RequireProject` with the empty action
   (may open the project) means a row with anything in it; doc checks
   (`Can`, `resolveDocWorkspace`) resolve the doc's project and apply the
   project layer before the doc overwrite; a doc in a project a restricted
   member cannot open is not found whatever the doc's overwrite allows. `HoldsAnywhere`,
   `PermissionsAnywhere`, and `RequireAnywhere` skip restricted memberships
   through a scopes query returning unrestricted workspaces only. Unrestricted
   members answer exactly as before.
4. **Projects** (`internal/workspace/usecase.go`): `List` returns only the
   projects the caller may open; `Get` and the board-setting reads move from
   workspace membership to `RequireProject`. `Create` needs `projects:write`
   with no project, so a Restricted member is refused. `DeleteImpact` gains
   the Restricted members who lose access (id, display name); `Delete`
   deletes the project's access rows in the same transaction.
5. **Membership** (`internal/tenancy`): the Owner can never be restricted.
   Use-cases to set Every project (`role` or `none`) and to replace one
   project's levels (empty removes the row), each needing `members:write` in
   the workspace and judged by what it adds: a giver sets only levels they
   hold on that project, and `role` or a project area in a role needs the
   levels with no project. Switching to `none` keeps stored rows; switching
   to `role` keeps them unused. Removing a member deletes their rows for that
   workspace's projects. A restricted member's role edits stay limited the
   same way (`internal/roles`).
6. **HTTP.** `GET /api/team` memberships gain `every_project` and
   `projects: [{project_id, project_name, allow}]`, listing only projects
   the viewer may open. `PATCH /api/workspaces/{workspaceID}/members/{userID}`
   gains `every_project` and `project_access: [{project_id, allow}]`, a
   patch like `allow` and `deny`. `GET /api/workspaces/{workspaceID}/me` adds
   `restricted` and, when on, `projects: [{project_id, actions}]`. New
   `GET /api/projects/{projectID}/access` (Restricted members with access and
   their actions; `members:write` in the workspace) and
   `GET /api/projects/{projectID}/people` (People who may open the project;
   anyone who may open it).
7. **Invitations** (`internal/tenancy/invitations_service.go`, `model.go`):
   a grant carries `every_project` and `project_access`; create checks the
   giver holds each level; redeem re-checks, skips deleted projects, writes
   `restricted` and the rows in the redeeming transaction, and never changes
   an existing membership. The preview shows the projects by name.
8. **Events.** Setting a project's levels publishes `access.grant.changed`
   (`resource_type: "project"`) through the outbox; widen the enum in
   `internal/integrations/catalog.go`, regenerate `sdk/src/events.generated.ts`
   (`bun run generate:events` in `sdk/`), add the `project` case to
   `grantScope` in `server/cmd/automation_scope.go`. Switching Every project
   publishes `workspace.member.updated`. In `server/cmd/live_audience.go`, the
   grant frame reaches the person and holders of `members:write` in the
   project's workspace.
9. **Pickers.** `tickets.SetPerson` refuses, as invalid, a developer or
   tester who may not open the ticket's project.
10. **Read-time filters.** `NotificationService.List` and `UnreadCount`
    leave out notices whose subject (ticket, doc, memory) the reader may not
    open; trail reads (`plays.Runner.GetTrail`, `ListTrails`, `trail_list`)
    leave out trails whose project the reader may not open. Rows are kept.
11. **Ticket attachments** (`internal/attachments/usecase.go`): a ticket
    owner's attachment is read, uploaded, and deleted through the ticket's
    `tickets:read` or `tickets:write`, not a bare signed-in check.
12. **Plays** (`internal/plays/run.go` `checkPlay`): confirm the target read
    goes through Project access, the excluded-projects check stays, and a
    play's excluded list keeps unseen project ids on update.

## Acceptance criteria

- [ ] Migration 0049 has an upgrade test from 0048 (`migrateBefore`):
      existing members and invitation grants read as From role, nothing taken
      away
- [ ] Table-driven resolver tests, error paths first: restricted member with
      no rows sees no project; project-area action with no project refused;
      instance-area action refused in-workspace and anywhere; doc overwrite on
      top of the project layer, and a doc allow in a hidden project still not
      found; Owner unaffected; unrestricted unchanged
- [ ] A restricted caller gets not found for a hidden project, ticket, doc,
      stack, deploy, its board settings, and its ticket attachments, through
      HTTP and the use-case
- [ ] `instance_permissions` on `/api/auth/me` ignores restricted memberships
- [ ] Granting a level the giver does not hold is forbidden; redeeming an
      invitation whose giver lost a level admits nobody
- [ ] Removing a member, deleting a project, and switching Every project both
      ways leave no stale access and lose no stored levels
- [ ] Notices and trails for a hidden project vanish from lists and unread
      counts and return when access is given back
- [ ] A live ticket frame stops reaching a socket the moment access is taken
- [ ] Integration test on real SQLite covers the end-to-end path
- [ ] Hit every surface: HTTP routes yes; MCP in ticket 11; events (catalog
      row, outbox, SDK) yes; live push yes; search inherits through
      `RequireProject`, verify; permissions yes; reverse states (None back to
      From role, access given back) yes; docs: `website/` projects and
      roles pages if they state who reads the project list
- [ ] `make lint`, `make coverage`, `make sqlc-check`, `make vuln`; `bun run
      typecheck` and `bun run test` in `sdk/`

## Read first

`practices/go.md`, `practices/architecture.md` (sections 1 to 6),
`practices/testing.md`, `practices/borrowed-practices.md`; ADRs 0042, 0061,
0085, 0087, 0088, 0097; `CONTEXT.md` (Restricted member, Project access,
Every project); this effort's `spec.md` and tickets 01 to 05.

## Files likely touched

- `internal/platform/storage/migrations/0049_*.sql`, `queries/`, a
  `migration_0049_test.go`
- `internal/platform/permissions/permissions.go`
- `internal/access/` (`usecase.go`, `gate.go`, `repo.go`, `handler.go`)
- `internal/workspace/usecase.go`, `model.go`, `handler.go`
- `internal/tenancy/` (usecase, handler, invitations, model, repo)
- `internal/roles/usecase.go`
- `internal/tickets/usecase.go`, `internal/attachments/usecase.go`
- `internal/plays/run.go`, `usecase.go`
- `internal/integrations/catalog.go`, `sdk/src/events.generated.ts`
- `server/cmd/live_audience.go`, `automation_scope.go`, `wire_gates.go`,
  `routes.go`
