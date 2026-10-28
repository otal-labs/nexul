# Change history: see what changed, like a git diff

Label: wayfinder:map

## Destination

A spec in `.scratch/change-history/spec.md`, with build tickets, for a history
on docs, tickets, plays, memories and automations: a list of past versions,
and a dialog showing what changed between two of them the way a git diff
does. When the spec is done, the building starts.

## Notes

- Domain: `internal/docs`, `internal/tickets`, `internal/plays`,
  `internal/memories`, `internal/automations`, and the web pages that show
  them. Vocabulary from `CONTEXT.md`.
- Already there: `doc_versions` (stored, nothing on screen shows it),
  `memory_versions` (a versions feed and revert, no diff), and
  `automation_versions` (a feed and a line diff, `web/src/utils/DiffUtility.tsx`).
  Tickets and plays keep no history.
- `.scratch/ticket-workflow-depth/` already specs a ticket activity timeline
  (user stories 15 and 16). The ticket history decision has to agree with
  that spec or replace it.
- Prototype and visual tickets go through `design-mode`. Web is built at
  768px first and checked at 768, 1024 and 1440 (AGENTS.md, rule 12).
- Every surface counts: HTTP route, MCP read, live push, permissions, the
  guide page (AGENTS.md, Hit every surface).

## Decisions so far

<!-- one line per resolved ticket: - [title](issues/NN-slug.md): gist -->

- [How versions are written today, and how big they get](issues/01-how-versions-are-written-today.md): docs version on API/MCP saves and snapshots, never on editor flushes; memories on every save; automations per push. The existing line diff holds to ~5,000 lines, and prose needs word-level highlights. Demo counts are seeded, so no real rates are known.

## Not yet specified

- One shared versions store for every entity, or a table per entity like
  today. Depends on what tickets and plays end up storing.
- How agents reach history over MCP within the tool budget: a `version`
  field on each `get`, a diff in the result, or nothing.
- Retention: whether frequent saves need squashing or pruning. The demo
  data is seeded, so real save rates are unknown; it hangs on how often the
  editor ends up writing versions.
- Live push: whether an open history list updates when someone else saves.
- Who sees history: the entity's read permission, or something stricter.
- The phone app: whether it shows history at all.

## Out of scope

- Agent find-and-replace edits on `doc_update`, `ticket_update`,
  `play_update` and `memory_update` (so an agent stops resending whole
  bodies). Already clear, so it ships as its own PR outside this map.
- Git-style branches, merges or conflict resolution between versions.
