# 02 — Which permission areas are project areas

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

Every domain in the permission table (`internal/platform/permissions`) has to
land in the role editor's Workspace section or its Every project section.
Which goes where, and what happens to the domains that live on both sides:
memories (workspace and project memories), plays (defined per workspace, run
on a project's ticket), the `projects` domain itself (creating a project is
a workspace act, editing and deleting one is a project act), board settings,
repositories and reviews, automations?

Produce the table with a recommendation per row and the reason, and walk the
owner through the rows that are not obvious.

## Answer

Settled with the owner, 2026-10-01.

| Every project (Project access levels) | Workspace (role only) |
|---|---|
| tickets, docs, memories, stacks, deploys, pull requests, code reviews, repositories, attachments, permissions (sharing a doc), projects and board settings | chat, channels, voice, notifications, mentions, plays, members and invites, roles, workspaces |
| | Instance areas: runners, machines, topology, DNS, connectors, integrations, automations, events, audit log, accounts, instance settings |

- **Memories are project-level only.** Workspace memories (ADR 0059) are
  removed: the "Workspace" destination leaves the Clone dialog, a memory
  always needs a project, and the migration deletes every workspace-scoped
  memory that exists. A plain chat with no ticket or doc carries no
  memories again. Needs an ADR superseding ADR 0059.
- **Projects keep one row.** `projects:write` still both creates a project
  and edits one; no create verb. Creating runs on the role's Every project
  levels, so a Restricted member never creates a project.
- **Permissions (who sees a doc) is a project area**, since a doc lives in
  one project.
- **Plays stay in Workspace.** Running one on a ticket needs `plays:run`
  from the role and the ticket readable through Project access.
- Stacks that belong to no project stay instance-level.
