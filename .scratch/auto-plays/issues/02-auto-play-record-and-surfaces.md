# 02: The auto play record, its permissions, and its HTTP and MCP surface

Type: research
Status: open
Blocked by: None — can start immediately

## Question

An auto play belongs to one play and holds: enabled, the moment, the
conditions (all/any groups nested one level, over ticket type, project,
stage, status column, category, labels, developer, tester, has a source
doc, has a linked PR, blocked; doc project and folder), priority rules,
limits (once per ticket per occurrence over a period), and run on
(developer, tester, whoever caused it).

- The table shape (columns versus one JSON column for conditions and
  priority rules), the migration, and what happens to an auto play when its
  play, a label, a project, or a status column it names is deleted.
- The new `autoplays` permission domain (`read`, `write`, `delete`) in
  `internal/platform/permissions`, its Access page entry, and its live
  audience rule.
- HTTP routes under the play, and the MCP shape: extend `play_*` (for
  example `auto_plays` on `play_update` and in `play_list`) within the tool
  budget in `practices/mcp.md`, or a new `autoplay_*` family.
- Events for an auto play created, updated, deleted, and the live push the
  settings page needs.
- Whether the workspace's daily per-ticket cap (default 5) and the
  per-person concurrency cap are workspace settings or instance templates.

Findings go in `research/02-auto-play-record-and-surfaces.md`.
