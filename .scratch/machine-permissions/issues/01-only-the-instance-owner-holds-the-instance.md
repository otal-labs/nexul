# 01: Only the instance Owner holds the instance

Type: task
Status: open
Blocked by: None — can start immediately

Read first: `AGENTS.md`, `CONTEXT.md` (Instance area, Instance Owner, Source), `practices/go.md` (sections 14, 17),
`practices/architecture.md` (sections 2, 3, 8), `practices/mcp.md`, `practices/testing.md` (sections 3, 8, 10), ADRs
0088, 0148, 0151, the spec (decisions 3 and 4).

## What to build

- `decideArea` (`internal/access/area.go`) stops treating an Owner of any workspace as holding every area: only the
  instance Owner (`IsInstanceOwner`) is allowed and never denied. An Owner of another workspace answers from their
  roles and the area's rules like anyone else.
- Every other instance-level check that reads "Owner of any workspace" follows: `RequireAnywhere`, `HoldsAnywhere`,
  `PermissionsAnywhere` and their callers for instance-area actions, the live audience's `anywhere`, and
  `/api/auth/me`'s `instance_permissions`. Workspace-area and project-area checks inside a workspace keep the Owner
  bypass.
- The forbidden answer's source sentence covers the case ("Only the instance Owner holds this unless a rule gives
  it").
- Update `CONTEXT.md` (Instance area: drop "An Owner of any workspace is never denied one") and the guide's
  `people-and-access.md`.

## Acceptance criteria

- [ ] `TestInstanceAreas_AnotherWorkspacesOwnerHoldsNoArea`, over HTTP and MCP, and with a rule that gives one area.
- [ ] `TestInstanceAreas_TheInstanceOwnerIsNeverDenied` still passes.
- [ ] The Settings entries the web shows follow `/me` (verified at 768, 1024 and 1440px as another workspace's Owner).
