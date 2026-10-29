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
- The ticket activity timeline from `.scratch/ticket-workflow-depth/`
  (user stories 15 and 16) is now part of this effort; that spec keeps only
  custom fields.
- Prototype and visual tickets go through `design-mode`. Web is built at
  768px first and checked at 768, 1024 and 1440 (AGENTS.md, rule 12).
- Every surface counts: HTTP route, MCP read, live push, permissions, the
  guide page (AGENTS.md, Hit every surface).

## Decisions so far

<!-- one line per resolved ticket: - [title](issues/NN-slug.md): gist -->

- [How versions are written today, and how big they get](issues/01-how-versions-are-written-today.md): docs version on API/MCP saves and snapshots, never on editor flushes; memories on every save; automations per push. The existing line diff holds to ~5,000 lines, and prose needs word-level highlights. Demo counts are seeded, so no real rates are known.
- [How many versions a history keeps](issues/07-how-many-versions-a-history-keeps.md): the last 10 unnamed versions per body history, named versions and live automation versions never pruned; ticket field changes kept separately, up to 100.
- [What a ticket's history is](issues/02-what-a-ticket-history-is.md): a list of changes (who, when, via which play or automation) over every field except position, with a text diff only for the description; starts on ship day; absorbs the old activity-timeline spec.
- [When editing a doc writes a version](issues/06-when-editing-a-doc-writes-a-version.md): one version per sitting, about 5 minutes after the last edit or when the last editor leaves, credited to everyone who edited; saves that change nothing write no version.

## Not yet specified

- One shared versions store for every entity, or a table per entity like
  today. Depends on what tickets and plays end up storing.
- How agents reach history over MCP within the tool budget: a `version`
  field on each `get`, a diff in the result, or nothing.
- Live push: whether an open history list updates when someone else saves.
- Who sees history: the entity's read permission, or something stricter.
- The phone app: whether it shows history at all.

## Out of scope

- Agent find-and-replace edits on `doc_update`, `ticket_update`,
  `play_update` and `memory_update` (so an agent stops resending whole
  bodies). Already clear, so it ships as its own PR outside this map.
- Git-style branches, merges or conflict resolution between versions.
