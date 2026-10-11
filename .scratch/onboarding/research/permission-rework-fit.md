# How the permission rework lands on roles, invitations and computers

Read from `origin/master` (06bc3014) and, where marked, `origin/permission-switch` (0adfd8cd, three commits ahead of
master). Spec: `.scratch/permission-overrides/spec.md`; ADR 0148.

## 1. The role after the rework

- **Table.** `roles(id, workspace_id, name, is_owner_role, permissions TEXT json, created_at, updated_at)`
  (`migrations/0001_schema.sql:307`). The rework adds no column to it. A role only allows; Deny lives in rules on
  things (spec, Roles). Owner stays one role with the bypass, singleton per workspace (`idx_roles_owner_singleton`).
- **Several roles.** `member_roles(user_id, workspace_id, role_id, created_at)` already exists on master
  (`migrations/0098_member_roles.sql`, #596). `workspace_members` lost `role_id` and keeps `restricted` until the
  switch. Roles combine as the union; zero roles is legal (no project, no play; chat, DMs and the inbox still work).
  Every writer still gives exactly one role on master (`ChangeMemberRole`, `internal/tenancy/usecase.go:475`).
- **ADR 0148 replaces the Restricted member.** Restricted and Every project go (ADR 0097 superseded). Projects become
  opt-in: a person is in a project only when its rules allow `projects:open` for a role they hold or for them by name.
  A Template role therefore gives no project access until a project lists it, which is the Access page's one-time grid
  (slice 13), not the role. `projects:open` never appears in the role editor.
- **New role bits.** `projects:create` (workspace) and `projects:access` (project), backfilled to holders of
  `projects:write` and `members:write`. The editor is a dialog with Instance, Workspace and In projects tabs (#595,
  slice 08 done); slice 15 renames "Every project" to "In projects" and adds the two switches.
- **Roles carry no UI state.** Nothing in `roles` or `member_roles` could hold ticked Welcome steps. They would need a
  new column or table (not checked against any open slice).
- **Instance bits.** A role bit for an instance area holds that area in any workspace the holder is in (ADR 0088; spec,
  Instance areas). A Template role should hold none.

## 2. Invitations after the rework

- **Master today.** `invitation_grants` has one `role_id` per workspace plus per-person `allow_json`, `deny_json`,
  `restricted`, `project_access_json` (`migrations/0002_private_invitations.sql:13`; `invitations.sql`,
  `CreateInvitationGrant`). `workspace_invites` has one `role_id` per login (`0001_schema.sql:333`).
- **After.** At least one role per workspace, no per-person parts (spec, Roles and MCP). `invitation_create` takes
  `roles`. Open links keep their role and lose the rest (slice 10 wipe).
- **Project access.** An invitation grants none. A person lands in no project until a project's rules list one of
  their roles (or them). Listing a role on a project needs Manage this project's access on that project.
- **Branch state (unverified in detail).** `origin/permission-switch` has slices 09, 10 and 11 marked done
  (`b41a322c`, `d33c3272`, `0adfd8cd`): `InvitationGrant` carries `RoleIDs`
  (`internal/tenancy/invitations_service.go:183`), migrations 0099 to 0102 (wipe, `access_setup_finished` on
  `workspaces`, `workspace_invites.role_id` now cascades on role delete). So deleting a role also deletes its pending
  invites, which matters if an admin deletes a Template role copy.

## 3. Instance templates today, and what a Template role needs

- **Storage.** `instance_templates(kind, key, body, updated_by, updated_at)`, primary key `(kind, key)`; a row exists
  only once edited, none means the code default (`migrations/0055_instance_templates.sql`,
  `queries/instance_templates.sql`).
- **Registration.** `server/cmd/wire_templates.go:31-50` registers a `templates.Kind` per kind with `Defaults`
  (key, name, body), `Below` scope (`workspace`, `project` or none), `Follows`, optional `Check` and `Questions`, and a
  `Layer` that reads, writes and resets below the instance (`internal/templates/model.go`). Bodies are text.
- **Copy at creation.** Plays and ticket types use `Follows: false`: copied when a workspace or project is created and
  never rewritten (ADR 0103). Plays are seeded by `SeedWorkspaceDefaults`, called from `tenancy.Create`
  (`internal/tenancy/usecase.go:91`; `server/cmd/wire_gates.go:321`), right after the Owner role and `#general`.
- **Permission.** Writing needs `templates:write` in any workspace; reading needs only sign-in (ADR 0103). Settings →
  Templates is `web/src/components/templates/InstanceTemplatesSection.tsx`, one group per kind, labelled in
  `web/src/models/Template.tsx`.
- **What a Template role needs.** A new kind (`Below: workspace`, `Follows: false`) with five default keys; body as
  JSON (name, permission list, ticked steps); a `Check` rejecting unknown permission keys; and a `Layer` whose
  Write/Reset act on the workspace's copy. Seeding at workspace creation would call `roles.Service.Create`
  (`internal/roles/usecase.go:61`, checks the actor holds what it gives; the creator is an Owner, so it passes).
  `roles.Service.Clone` (`:199`, needs `roles:clone`, refuses the Owner role) is the closest existing copy. Existing
  workspaces get nothing, matching how plays were handled.
- **Rework overlap.** The branch already edits `templates_integration_test.go`, `InstanceTemplatesSection.test.tsx` and
  `Template.tsx` (migration 0102 adds `memories:template`), so a new kind there lands next to its changes.

## 4. Personal runners and "Your computer"

- **Built on master** (tickets 01 to 10, 12, 13, 17, 20; 11 is wontfix): enrolment by one install command, pairing
  through the runner, the runner minting the pairing token, T3 Code install and keep-alive, computer facts, the Add a
  computer dialog (#566), macOS and Windows installs, container logs, Cloudflare optional in the owner wizard (#574).
- **Not built.** 14 (remove tunnel and URL pairing; waits 30 days from the move notice), 15 and 16 (shell jobs), 18 and
  19 (sharing a computer; wait for rework slice 21, still open). The spec's build order says 15 and 16 wait for rework
  slice 01, which is done.
- **A computer row** (`internal/pairing/model.go:17`): `UserID` owner, `Kind`, `Name`, `ServerURL`, `TokenExpiresAt`,
  `SetupConfirmedAt` (ADR 0063), `Runner *ComputerRunner` (nil until one enrols), `PairError`, `Facts`/`FactsAt`
  (owner-only: OS, T3 state, providers, Docker, git identity), and `Tunnel *ComputerTunnel`.
- **A tunnel-paired computer** has `Tunnel` set and `Runner` nil. The next facts report after a runner enrols pairs it
  through the runner and then deletes the tunnel (ticket 12, `internal/pairing/move_usecase.go`). Until then it works
  but has no facts, and its owner has one `computer.move_requested` inbox notice. "A computer exists" is true for
  either shape; the setup checks (T3 answering, providers signed in) are only reported from a runner's facts.
- **A computer belongs to no workspace** and only its owner sees it (ADR 0146; spec, Computers). Rules on it name
  people only (slice 21, open).

## 5. Open rework tickets

- **Done on master:** slices 01 to 08 (sources, role rules, role sets, project rules read for everyone, channel rules,
  panel on docs and plays, row menu, role editor dialog). All rework map tickets are resolved.
- **Open:** 09 to 24. On `permission-switch`, 09, 10 and 11 are done (status lines only, not verified). Still open: 12
  several roles on screen (Team rows, person and invitation dialogs), 13 Access page and setup, 14 People in this
  project, 15 role editor sections, 16 client-core and the phone, 17 box upgrade test and the switch PR. Then 18 to 24
  on master (folder, stack, channel rules, computer sharing, See on plays, instance areas).
- **Files shared with this effort.** While the branch is open nobody else works in `internal/access`,
  `internal/tenancy`, the channel reads in `internal/chat`, or `web/src/components/{access,team,member,settings}`
  (spec, Build and release). The Welcome wizard touches tenancy (workspace creation seeding, step progress), Team
  (progress per person; slice 12), invitations (slices 10 to 12), the role editor (a steps control; slices 08 and 15),
  and `web/src/components/templates/`. Slices 12 and 13 both rewrite the guide page `people-and-access.md`.
- **Release effect.** The switch wipes every per-person permission and lands as one squash PR after the box test (17).
  Nothing in it changes who holds a Welcome step, since steps hang off roles that survive the wipe.
