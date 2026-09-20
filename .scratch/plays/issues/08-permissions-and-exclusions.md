# 08 — Permissions and exclusions

**Type:** grilling
**Status:** resolved
**Blocked by:** 01

## Question

1. The new permission rows in the domain table: `plays` with
   `read`, `write`, `delete`, and `memories` with `read`, `write`. Does
   running a play need only `plays:read`, or a separate `plays:run`? The
   vocabulary is read/write/delete only (ADR 0010); confirm read means
   "see and run".
2. User exclusion as a permission overwrite: overwrites exist for docs and
   the workspace (`resourceTypeDoc`, `resourceTypeWorkspace` in
   `internal/access/usecase.go`). Adding `play` as a resource type: where in
   the UI does the owner deny a user one play (on the play's settings page,
   reusing the doc permissions dialog)?
3. Project exclusion: a list of project ids on the play. Does an excluded
   project also hide the play from MCP `play_list` for that project?
4. Which roles get `plays:*` and `memories:*` by default in a fresh
   workspace, and does the migration grant existing roles that have
   `docs:write` the new `memories:write`, since toggling a memory used to
   need only that?
5. Copying a memory to another project or workspace: `memories:write` in
   the destination, and membership of the destination workspace, since a
   workspace is an isolation boundary. Confirm, and whether the copy carries
   the source's attachments.
6. Doc threads carry no permission of their own (ticket 04): the chat
   use-case checks the doc's read bit for `doc_thread` conversations the
   way it checks whatever it checks for ticket threads today. Confirm the
   ticket-thread precedent and that MCP `chat_list_messages` honours it.
7. A run acts with the starter's permissions on the harness. If the play's
   move-to needs `tickets:write` and the starter lacks it, is the button
   disabled up front, or does the run succeed and the move fail with a note?

## Answer

Resolved 2026-09-16 with the owner (two grilling rounds).

- **The permission vocabulary gains domain-declared verbs.** Read, write,
  and delete stay universal; a domain may declare a verb when the act is
  neither reading nor editing. Three arrive with this effort: `plays:run`,
  `memories:clone`, `docs:thread`. Recorded as an amendment to ADR 0010 in
  ticket 10; the Go domain table already lists actions per domain, so the
  catalog, roles grid, and token scopes pick them up.
- **Plays**: `plays:read` sees the definitions list in settings;
  `plays:run` sees the button on tickets and docs and fires it;
  `plays:write` creates and edits; `plays:delete` deletes. Only `run` shows
  buttons; only `read` shows settings.
- **Memories**: `memories:read` sees the page and the picker,
  `memories:write` creates and edits, `memories:clone` copies to another
  project or workspace (plus membership of the destination workspace; the
  copy carries attachments as new rows). Nobody clones without the bit.
- **User exclusion** is a permission overwrite with resource type `play`,
  denying `plays:run` on one play, managed from the play's settings page
  with the doc sharing dialog generalised to a resource type.
- **Project exclusion** hides the play from the page and from the MCP
  list for that project.
- **Doc threads**: a doc's thread, its messages, the Agent's replies, and
  the play runs on it are visible and postable only with `docs:thread` on
  that doc, checked in the chat use-case so gateway and MCP agree. Granted
  per role and overridable per doc through the existing overwrites, so a
  client role can hold `docs:read` and see only the end result. Ticket
  threads stay member-open per ADR 0023.
- **No migration grants.** The owner has not deployed publicly and intends
  to restart the schema; a fresh workspace seeds the Owner role only, and
  every new role ticks its own bits.
- **A clicker without `tickets:write`** sees the move-to field disabled
  with "you can't move tickets in this project"; the run proceeds with no
  move.
