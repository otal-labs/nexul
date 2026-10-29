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
- [What a play version captures](issues/03-what-a-play-version-captures.md): instructions as body versions (last 10), other settings as field changes (100); existing plays start with their current instructions; each trail records the version it ran.
- [Restore from history](issues/04-restore-from-history.md): restore text (docs, ticket descriptions, play instructions, memories) as a new version on top; field changes never undone from history; a restore replaces live text after saving the open sitting; edit permission and the lock apply.
- [One versions store, or one per entity](issues/08-one-versions-store-or-one-per-entity.md): a table per entity, cascading on delete; one shared route shape and one web dialog; automation versions stay as they are.
- [Who reaches history, and where](issues/09-who-reaches-history-and-where.md): anyone who can read an item sees its history; agents read and revert through a `version` on each get and `revert_to_version` on each update, as memories do; an open list doesn't update live; no history on the phone.

## Not yet specified

- The "what changed" dialog's look and its diff (ticket 05).

## Out of scope

- History on the phone app (ticket 09).
- Agent find-and-replace edits on `doc_update`, `ticket_update`,
  `play_update` and `memory_update` (so an agent stops resending whole
  bodies). Already clear, so it ships as its own PR outside this map.
- Git-style branches, merges or conflict resolution between versions.
