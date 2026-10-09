# Restricted members see only the projects they hold Project access to

ADR 0087 made the project list something every member reads, so the only way to keep a client to one project was a
workspace of their own, away from where the team works. A client giving feedback on one OTAL project belongs in OTAL
and must not see OTAL's other projects, names included.

Decision: a membership may be restricted. A Restricted member sees only the projects they hold Project access to;
every other project is invisible, name included, and a direct link reads as not found. Projects created later stay
hidden from them. The Owner is never restricted. In the UI it is the Every project row on a person's workspace
access: From role (the role's project areas on every project, every existing member after the upgrade) or None (a
Restricted member, with a level per project area on each project given).

- **Every domain has an area.** Project: tickets, docs, memories, stacks, deploys, pull requests, code reviews,
  repositories, attachments, doc sharing, projects and board settings. Workspace: chat, channels, voice,
  notifications, mentions, plays, members and invites, roles, workspaces. Instance: runners, machines, topology, DNS,
  connectors, integrations, automations, events, audit log, accounts, instance settings. The permission table carries
  the area and the catalog serves it, so the role editor splits into Workspace and Every project from the server.
- **The resolver**, after the Owner bypass, for a restricted membership: a workspace-area action answers from the role
  and workspace-wide overwrite; an instance-area action is refused; a project-area action answers from that project's
  access alone and is refused with no project, so a Restricted member never creates a project; may-open-this-project
  means holding any access there; a doc's own overwrite applies on top of the project's level. For a Restricted
  member Project access is the whole answer inside a project: the role adds nothing.
- **Storage.** `workspace_members.restricted`, and Project access as `permission_overwrites` rows with
  `resource_type = "project"` and the levels in `allow` (ADR 0042's one table). Rows are read only while the member
  is restricted, so switching to From role and back loses nothing. Deleting a project deletes its rows.
- **Instance level.** `HoldsAnywhere` and `PermissionsAnywhere` skip restricted memberships, so a restricted
  membership opens no instance area, whatever its role carries.
- **Granting.** Anyone with `members:write` sets it from Team, and invitations carry it, re-checked at redemption.
  Nobody grants a level they don't hold on that project; From role and a role's project areas need the levels
  without a project, which only an unrestricted giver holds.
- **Live.** A Project access change publishes `access.grant.changed` with `resource_type: "project"`; switching the
  row publishes `workspace.member.updated`. The live socket re-checks every frame, so a project's frames stop the
  moment access goes.

People stay visible (ADR 0086): names are not the leak, project names and contents are.

The trade-offs: a restricted member's lists are filtered by a check per row, fine for one client and batched per
person if it grows. Project access holds levels only; there is no deny layer per project, so adjusting one project
for a From role person is done by switching them to None. Instance areas answer to unrestricted memberships alone,
so a client given a powerful role in their restricted workspace still cannot reach a runner.

Rejected: a workspace per client, which splits the team's work; implying restriction from holding any access row,
which shows a freshly invited client with no project everything instead of nothing; and a table of its own for
Project access, which would have repeated the overwrite repo, its cascade, and its event.

Amends ADR 0087: the project list and a project's own read take Project access for a Restricted member, and
instance-level areas count unrestricted memberships only. Amends ADR 0088 the same way for instance bits. Amends
ADR 0042: a project layer sits between the workspace and the resource, and for a Restricted member it replaces the
role and workspace-wide layers on project areas. Amends ADR 0085, whose Team dialog sets Every project and Project
access, and ADR 0061, whose invitation package carries them. Decided 2026-10-01. Amended by ADR 0135: a restricted member's
Project access is read once per request, not once per row.
