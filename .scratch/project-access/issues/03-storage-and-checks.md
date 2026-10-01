# 03 — How project access is stored and checked

**Type:** grilling
**Status:** open
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
