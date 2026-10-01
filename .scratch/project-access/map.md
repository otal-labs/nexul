# Wayfinder map: Project access and private channels

**Label:** wayfinder:map

## Destination

A spec with the ADRs it needs, ready to slice into implementation tickets:
a workspace member can be held to the projects they are given access to,
with what they may do set per project, and channels can be private. It
covers the role editor split, the Team dialog and project settings, MCP,
and live updates.

The case that started it: a client giving feedback on one project is added
to the OTAL workspace and must not see any of OTAL's other projects, not
even their names.

## Notes

- Domains: `internal/access` (the resolver, `RequireProject` in `gate.go` is
  the seam every project-scoped check goes through), `internal/tenancy`
  (memberships, Team), `internal/platform/permissions` (the domain table),
  `internal/chat` (channels), `server/cmd/live_audience.go`,
  `web/src/components/settings` (Team, roles).
- Skills every session should consult: `/grilling` and `/domain-modeling`
  for grilling tickets, `design-mode` for the prototype ticket.
- Read `practices/` per `AGENTS.md` before any code, including prototypes.
- Standing preferences from the owner, treat as fixed:
  - Roles are what someone does in the workspace; Project access is what
    they do in a project.
  - Hidden by default: a Restricted member sees a project only once given
    access, and new projects stay hidden from them.
  - Hidden means invisible, name included; a direct link reads as not found.
  - Every permission change reaches open clients in real time through
    events. Doc and play grants already publish `access.grant.changed` and
    the live socket re-checks every frame; project access joins that path,
    never a refresh-only version.
  - Production data is live: any schema change is a new numbered
    forward-only migration, and existing members upgrade as "All projects"
    with nothing they could do taken away.
- ADRs this effort will need: Restricted members and Project access
  (amends ADR 0087's "the project list takes membership only" and ADR
  0094's "reading a channel takes membership alone"), and private channels
  if ticket 04 lands somewhere surprising.

## Decisions so far

- [Scope and ground rules](issues/01-scope-and-ground-rules.md) — per-person
  "All projects" or "Only these projects"; Project access carries a level per
  project area; the role editor splits into Workspace and Every project;
  restricted members get no instance-level areas and no workspace channels
  except private ones they are in; set from the Team dialog.

## Not yet specified

- What an open page shows the moment its project is taken away from the
  person looking at it (the frames stop; does the page fall to not found,
  bounce to the board, or say why).
- Notices, inbox entries, and agent trails created before someone was
  restricted that point at a project now hidden from them.
- How a play's excluded-projects list and a restricted member's Project
  access meet when they run a play.

## Out of scope

- Adjusting one project for an "All projects" person (say, denying Sam
  deletes on one project). Switching them to "Only these projects" covers
  it; per-project tweaks for everyone else are a separate effort.
- Filtering runners, machines, or the topology canvas per project. A
  restricted membership opens no instance-level area at all.
- Hiding people's names. Every member still reads the workspace's people
  (ADR 0086); project names and contents are the leak, names are not.
