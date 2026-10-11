# Wayfinder map: permission rework

Charted 2026-10-11 with the owner. The permission overrides spec (`.scratch/permission-overrides/`, ADR 0148,
written 2026-10-10) layered three-state rules over today's model and promised that nobody's access would change.
Reading it back, the owner decided the model underneath is the mess: "Every project", Restricted members, an
Everyone target and per-person rows in four shapes. This map finds the way to a clean model that is roles only,
with opt-in projects, several roles per person and a wiped start, and takes that through to a rewritten spec.

## Destination

`.scratch/permission-overrides/spec.md` and its tickets rewritten to the decisions here, with ADR 0148 and
`CONTEXT.md` to match, and every screen's look locked. Another session then builds it start to finish without
asking the owner anything. This map plans; it builds nothing.

## Notes

- Grilling tickets: call the Skill tool for `grilling` and `domain-modeling`. `CONTEXT.md` is the glossary.
- Prototype tickets: `design-mode` with `prototype`. The owner picks by looking, rendered in the thread with
  `html_render`. Lock picks with `onik-save-design`, outside the repo.
- Research tickets write findings to `.scratch/permission-rework/research/`.
- Ask the owner in plain language, never with ticket or spec codes.
- Prod facts (2026-10-11, read-only): 7 workspaces, 8 projects, 20 memberships. Roles are Owner, Administrator,
  Developer, Senior Developer and Client. Five people hold Client and one of them is Restricted. There are 32
  doc rows and 2 project rows in `permission_overwrites`.
- A first draft that applies the 2026-10-11 morning decisions is on the local branch
  `spec/permission-opt-in-projects` (`d0580b18`, worktree `nexul-perm-spec`, not pushed). Charting overturned
  parts of it: no Everyone, a wipe instead of preserving access, several roles, zero roles and a See permission
  per play. Use it as raw material for the rewrite, not as decisions.

## Decisions so far

- [The access model after the rework](issues/01-the-access-model.md): roles only, with no Everyone. A person
  can hold several roles or none. Projects are opt-in by role or person, and channels and plays stay
  workspace-wide with rules. Per-person rows are wiped on upgrade (a beta, on every install), except a doc
  creator's own rights. Owners fill a one-time "who's in each project" grid.
- [What the rework touches in the code](issues/02-what-the-rework-touches.md): every member gets the project read
  that only Restricted members had. One role is assumed across schema, queries, invitations, `/me` and MCP, with
  no tool headroom. Restricted also hides public channels. On production the wipe deletes 2 rows and keeps the 32
  creator rows.
- [The permissions panel and the right-click menu](issues/03-the-permissions-panel.md): a dialog with the targets
  in a column (a scrolling tab row at 768px) beside each permission as a compact Allow / Fallback / Deny select, the
  inherited answer and its source under each Fallback, and Check access in the same dialog. A project fronts it with
  a "People in this project" card, and Permissions… joins every row menu, right-click or ⋯, just above Delete.
- [Several roles on screen, and a person with none](issues/04-several-roles-on-screen.md): a roles checklist with a
  reasoned "Can't give" group, role chips in Team rows, and an empty state naming the Owners for a member with none.
- [The one-time "who's in each project" setup](issues/05-the-one-time-project-setup.md): one page with a projects ×
  roles tick grid ("Every project" lead row, People column), then a public channels grid ticked "can see", and one
  Finish. A non-Owner gets an empty state until then. Afterwards it is the workspace's Access page, `/:workspace/access`.
- [Build order and the release that wipes](issues/06-build-order-and-release.md): access-neutral prep (sources,
  role targets, role sets, batched project and channel reads, the doc and play panel, the row menu) lands on master
  first; everything that changes access is built on a short `permission-switch` branch and lands as one squash PR
  after a box upgrade test; folders, stacks, channel rules, play See, computers and instance areas follow.
- [The role editor as a dialog](issues/08-the-role-editor-dialog.md): Edit and New role open a 40rem dialog with a
  pinned Save; Workspace and In projects sections each lead with their own bulk row, Create projects and Manage this
  project's access are switches, plays' first rung reads See, extra verbs sit as checkboxes under their area, and
  leaving with changes asks first.
- [Rewrite the spec and tickets](issues/07-rewrite-the-spec.md): `.scratch/permission-overrides/` rewritten to
  these decisions as 24 slices, with ADR 0148, the ADRs it supersedes and `CONTEXT.md` to match. The map is complete;
  keep this directory until the slices that build its locked looks merge.

## Not yet specified

- How the phone app shows a project or play that disappeared, and the new pill states. It probably follows the
  web, once the web looks are locked.
- Whether "Request access" on a locked doc pill (turned down for now) comes back as its own effort.
- Statement counts and timings for the opt-in project check on a production copy: the spec's Performance plan sets
  the targets, and slices 02, 04 and 09 report the numbers.

## Out of scope

- Token scopes (held at the gateway, ADR 0087) and time-limited rules, as in the original spec.
- Preserving anyone's access across the upgrade: the owner chose a wipe (ticket 01).
