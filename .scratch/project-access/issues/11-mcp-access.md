# 11 — Project access over MCP and in the server instructions

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 08
**Decided in:** ticket 06; spec section "MCP"; ADR 0097

## What to build

Expose ticket 08's use-cases to agents. No tool is added; the ceiling does
not move.

1. `account_list` (`internal/mcp/composite/accounts.go`): each membership
   gains `every_project` (`role` or `none`) and, under `none`, `projects:
   [{project_id, project_name, allow}]`, listing only projects the caller
   may open.
2. `account_update`: a `workspaces` entry gains `every_project` and
   `project_access: [{project_id, allow}]`; each named project's levels are
   replaced, an empty `allow` removes the project, omitted fields keep their
   value. Steps run in the documented order and stop at the first failure
   naming the field, as today. The description says who may set it and that
   nobody grants a level they don't hold.
3. `invitation_create` (`internal/tenancy/invitations_mcp.go`): a `grants`
   entry gains the same two fields; `invitation_list` shows them.
4. `project_get` (`internal/mcp/composite/projects.go`): gains `access`, the
   Restricted members with access and their actions, filled only for a
   holder of `members:write` in the workspace, empty otherwise;
   `delete_impact` already carries the Restricted members who lose access
   from ticket 08, so its description says so.
5. `role_update` (`internal/roles/mcp.go`): its description says a role's
   project areas reach only members whose Every project is From role.
6. Server instructions (`internal/mcp/instructions.go`): one line saying a
   person may see only some of a workspace's projects, so not found can
   mean no access, not a fault. Stay under 2,048 characters.
7. The docs site's MCP guide
   (`website/src/content/docs/docs/guide/mcp-server.md`) where it lists
   these tools.

## Acceptance criteria

- [ ] Tests through `Call`, error paths first: invalid `every_project`;
      unknown project; forbidden actor; a level the caller does not hold;
      the patch rule (omitted `project_access` survives)
- [ ] `project_get.access` is empty for a caller without `members:write`
- [ ] The surface test (`internal/mcp/surface_test.go`) passes: descriptions
      of three sentences or more, every parameter described, instructions
      under the limit and naming only real tools
- [ ] A restricted token's `project_list` returns only its projects (live
      check against a dev server, per `practices/mcp.md` section 10)
- [ ] Hit every surface: MCP yes; HTTP in ticket 08; permissions yes; docs
      yes

## Read first

`practices/mcp.md` (all, sections 9 and 11 especially), `practices/go.md`,
`practices/testing.md`; ADRs 0068, 0081, 0085, 0097; this effort's
`spec.md` and ticket 06.

## Files likely touched

- `internal/mcp/composite/accounts.go`, `projects.go`, their tests
- `internal/tenancy/invitations_mcp.go`
- `internal/roles/mcp.go`
- `internal/mcp/instructions.go`, `surface_test.go`
- `website/src/content/docs/docs/guide/mcp-server.md`
