# Change history: see what changed, like a git diff

**Status:** ready-for-agent

## Problem

Docs keep versions nobody can see. Memories show a version feed with no diff,
and tickets and plays keep no history at all, so "who changed the
acceptance criteria" or "what did this play say last week" has no answer. The
versions that are kept are written at the wrong moments: every API save of a
memory writes one, even an unchanged save; a doc edited in the live editor
writes none.

## Decisions (owner, 2026-10-29)

The map's tickets hold the reasoning; this is what they settled.

1. **Docs, ticket descriptions, play instructions and memories keep body
   versions**, each compared word by word with the version before it.
   Automations keep `automation_versions` as it is (ticket 08).
2. **Tickets and plays also keep field changes**: one row per changed field,
   who made it and through which play or automation ("Alice, via Implement
   with AI"). Recorded on tickets: title, status, category, type, developer,
   tester, due date, labels, links, the doc it was filed from, the finished
   date. On plays: label, description, enabled, the stage it shows at,
   excluded projects. Never position or internal fields (tickets 02, 03).
3. **The live editor writes one version per sitting**: once the text has had
   no edits for 5 minutes, or when its last editor leaves, credited to
   everyone who edited. Live updates and background saves are unchanged
   (ticket 06).
4. **A save that changes nothing writes no version**, from any path: editor,
   HTTP, MCP, memory saves (ticket 06).
5. **Keep the last 10 unnamed versions per body**, and the last 100 field
   changes per ticket or play. Named versions and an automation's active and
   pending versions are never pruned (ticket 07).
6. **Restore writes the old text as a new version on top**, for docs, ticket
   descriptions, play instructions and memories. It saves the open sitting
   first, then replaces the live text the way an API save does (ADR 0124).
   Field changes are never undone from history. Restore needs edit
   permission; a locked doc refuses it (ticket 04).
7. **A table per entity**, cascading on delete; one route shape and one web
   dialog over all of them (ticket 08).
8. **Anyone who can read an item sees its history.** Agents read and restore
   it through a `version` on each get and `revert_to_version` on each update,
   as memories already do; no new tools. An open history list does not
   update live. The phone shows no history (ticket 09).
9. **One History dialog**: rendered text with added words tinted green,
   removed words struck through in red, and a margin bar per changed
   paragraph (grey edited, green added, red removed); a "Show changes"
   switch; each version against the one before it; the list on the right
   holds versions and, on tickets and plays, field changes as rows that
   can't be selected. The word diff is written in the web app, no new
   dependency (ticket 05). Reference mocks are outside the repo with the
   locked design references.
10. **History starts on ship day.** Existing tickets get one "Created by"
    change from their stored reporter and creation time; existing plays get
    their current instructions as version 1, an unedited built-in marked
    "Built-in default". Nothing is rebuilt from old events.

## The route shape

Every entity with a history answers the same two reads, under its own
prefix, gated by the entity's read permission:

- `GET /api/<entity>/{id}/history` returns `{ versions, changes }`, newest
  first, with no bodies. A version is `{ version, created_at, name, editor_ids,
  via, current }`; a change is `{ id, field, from, to, actor, created_at }`,
  where `actor` is the ticket `Actor` shape (kind, user id, play label and
  trail id, or automation id and name). Entities without field changes return
  `changes: []`.
- `GET /api/<entity>/{id}/versions/{n}` returns one version with its body.

Restore is `POST /api/<entity>/{id}/versions/{n}/restore`, gated by the
entity's write permission, and lands in the audit log like every non-GET
route. Docs already have the version read; `GET /api/docs/{id}/versions` stays
for its existing callers until nothing uses it. Automations answer
`/history` from `automation_versions`, sequence as the version number, and
keep their merge and rollback routes.

The dialog fetches `/history` when it opens, then the picked version and the
one before it. Two bodies at a time, never the whole history.

## Storage

Additive migrations, one per slice, each asked for in its ticket's PR:

- `doc_versions` gains `editor_ids TEXT NOT NULL DEFAULT '[]'` and
  `via TEXT NOT NULL DEFAULT ''`; `author_id` stays the first editor.
  `memory_versions` gains `editor_ids` the same way (`author_via` is its via).
- `ticket_description_versions (ticket_id → tickets ON DELETE CASCADE,
  version, body, editor_ids, via, created_at, PRIMARY KEY (ticket_id,
  version))`.
- `ticket_changes (id, ticket_id → tickets ON DELETE CASCADE, field,
  from_value, to_value, actor JSON, created_at)` with an index on
  `(ticket_id, created_at DESC)`.
- `play_instruction_versions` and `play_changes`, the same two shapes keyed
  by `play_id → plays ON DELETE CASCADE`. `play_trails` gains
  `instructions_version INTEGER` (null for runs before ship day).

A version or change is written in the same transaction as the save that
causes it, followed by one prune statement that deletes the oldest unnamed
rows past the cap. `via` is the automation name when an automation made the
save, `mcp` when an agent made it over MCP (the memory precedent, ADR 0049),
and empty for the browser. A play run's own name can't be recorded yet: its
agent calls MCP with the person's computer token, which names no run
(ticket 20).

## The sitting

`internal/collab` already knows who sent each update (`Store.AppendUpdate`
takes the actor). A room collects the editors since its last version and
holds one timer, reset by every update. When the timer fires (5 minutes) or
the last writer leaves, the room calls a new `DocWriter` method,
`CommitSitting(ctx, id, editorIDs)`, which writes a version of the current
body if it differs from the last one. Docs and ticket descriptions both
implement it; ticket descriptions use the same rooms. A restore or API save
that resets a room calls the same path first, so the sitting is never lost.
The timer is tested with `testing/synctest`, never a sleep.

## The diff

`web/src/utils/DiffUtility.tsx` grows a `diffWords` beside `diffLines`. Both
skip the matching run at the start and end before building the LCS table, so
a small edit in a long text builds a small table. The dialog turns each
version's Tiptap JSON into blocks (paragraph, heading, list item, code
block), diffs blocks, then words inside a changed block, and renders the
blocks with the editor's read-only styles plus `ins`, `del` and the margin
bar. Marks the markdown export drops (underline, highlight) are not compared.
A guard test counts the LCS cells built for a one-word edit in a
5,000-paragraph doc, so the trim can't quietly go away.

## Surfaces

- **Web:** the History dialog, opened from the item's menu on a doc, a
  ticket, a play, a memory and an automation. 768px first, checked at 768,
  1024 and 1440. A trail shows "Ran with version 7", which opens the play's
  dialog at that version, or "Version 7 is no longer kept".
- **MCP:** `doc_get`, `ticket_get` and `play_list` (with an id) take
  `version` and list the recent versions and, for tickets and plays, the
  recent changes; `doc_update`, `ticket_update` and `play_update` take
  `revert_to_version`, sent alone. `memory_get`'s versions list drops from 50
  to the 10 kept. No new tools.
- **Events:** none new. A restore is a save and publishes what a save
  publishes; a history list doesn't update live.
- **Permissions:** the entity's read for history, its write for restore.
- **Docs:** `CONTEXT.md` gains History, Version, Sitting and Change, and the
  memory entry's "every save appends a version" changes to "every save that
  changes something". A guide page under `website/.../guide/` covers history
  and restore, grown by each web ticket.
- **Phone:** nothing.

## Out of scope

- Agent find-and-replace edits on `doc_update`, `ticket_update`,
  `play_update` and `memory_update`; their own PR.
- Comparing two arbitrary versions; branches, merges, conflict resolution.
- History on the phone.
- Undoing a field change from history.
