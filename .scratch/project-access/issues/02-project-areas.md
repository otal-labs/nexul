# 02 — Which permission areas are project areas

**Type:** grilling
**Status:** open
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
