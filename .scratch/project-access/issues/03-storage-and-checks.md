# 03 — How project access is stored and checked

**Type:** grilling
**Status:** resolved
**Blocked by:** 02

## Question

Where do "All projects or Only these projects" and a person's Project access
live, and how does the resolver answer with them?

- The mode: a column on the membership, or implied by holding any Project
  access row.
- Project access: rows in `permission_overwrites` with
  `resource_type = "project"` (ADR 0042's one table), or a table of its own.
- The order of checks inside `HasPermission` and `RequireProject` for a
  Restricted member, and how a doc's own overwrite sits on top of the
  project's level.
- Filtering the project list and every cross-project list (search, mentions,
  inbox, board counts) without a check per row where a join will do.
- Instance-level checks (`HoldsAnywhere`, `PermissionsAnywhere`) skipping
  restricted memberships.
- The event: `access.grant.changed` gains `project`, and the mode change
  rides `workspace.member.updated`; which web query keys each invalidates.
- The migration and how an upgrade is tested from the previous schema.

Technical: decide it, record why, and let the owner react rather than grill
each line.

## Answer

Decided 2026-10-01; technical, for the owner to react to.

- **The mode is a column**, `workspace_members.restricted`, default off, so
  every existing member upgrades as "All projects" in one forward-only
  migration. Not implied by holding access rows: a Restricted member with no
  project yet (just invited) must see nothing, not everything. The Owner can
  never be restricted, like every other change to the Owner.
- **Project access rides `permission_overwrites`** (ADR 0042's one table)
  with `resource_type = "project"`: `allow` holds the levels expanded to
  actions, `deny` stays empty. The repo, the user-delete cascade,
  `ListByResource` for project settings' "who has access" list, and the
  grant event come for free. Deleting a project deletes its rows, as
  deleting a doc does. Rows are only read while the member is restricted;
  what switching mode does to them is ticket 05's call.
- **Each domain row in `internal/platform/permissions` gains its area**
  (project or workspace, per ticket 02), and the catalog serves it, so the
  web editor's two sections render from the server.
- **The resolver**, after the Owner bypass, for a restricted member:
  - a workspace-area action answers from the role and workspace-wide
    overwrite, as today;
  - a project-area action answers from that project's access row alone, and
    is refused when asked with no project (so creating a project, which is
    `projects:write` in the workspace, is refused);
  - "member of this project" (the project list, a project's page, the
    project-scoped live frames) means holding a row with anything in it;
  - a doc's own overwrite still sits on top: doc checks resolve the doc's
    project and apply the project layer before the doc layer.
  `RequireProject` is the seam: tickets, docs, deploys, stacks, repositories,
  reviews, mentions, and the live audience already route through it.
  Memories move from a workspace check to `RequireProject`, since they are
  project-only after ticket 02. The project list and a project's own read
  move from workspace membership to the project check.
- **Cross-project lists** (search, the @ picker, the board, labels, inbox,
  PR counts) already drop rows the caller cannot read, one check per row
  through `RequireProject`, so they inherit the rule with no change. Kept
  per row: batching per person is the lever if a restricted member's lists
  get slow, the same one ADR 0087 named for live frames.
- **Instance-level checks** (`HoldsAnywhere`, `PermissionsAnywhere`, and so
  `instance_permissions` on `/api/auth/me`) skip restricted memberships,
  through a scopes query that returns only unrestricted workspaces.
- **What the client is told.** `/api/workspaces/{id}/me` adds `restricted`
  and, when it is on, each granted project's actions, so the web hides what
  the server refuses.
- **Events.** A Project access change publishes `access.grant.changed` with
  `resource_type: "project"` (catalog enum widened); a mode change publishes
  `workspace.member.updated`, which exists. Both invalidate the web's
  projects, board, tickets, docs, memories, and workspace-me queries; the
  live socket already re-checks every frame, so the frames stop the moment
  access goes.
- **Tokens and agents** act as their creator and are held to the creator's
  grant (ADR 0087), so they inherit the limit with no change.
- **Migrations**, numbered and forward-only, each with an upgrade test from
  the previous schema: one adds `restricted`; one deletes workspace-scoped
  memories (ticket 02) and their versions and attachments. The use-case
  refuses a memory with no project; no table rebuild for a NOT NULL.
- Chat's side (what a restricted member reads among channels) waits on
  ticket 04.
