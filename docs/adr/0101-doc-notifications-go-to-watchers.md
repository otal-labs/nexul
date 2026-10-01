# Doc notifications go to watchers

Every doc create and every save notified every member of the doc's workspace. On a busy workspace the inbox filled
with docs a person never opened, and the collapse of repeated unread rows only hid how much of it there was.

Decision: a doc's change notifications go to its watchers, the people who created it, edited it, or chose to watch it.

- The creator becomes a watcher when the doc is created, and anyone who saves an edit to its title or body becomes one
  on that save, in the same transaction. The person the save is attributed to is the one added, so an edit made over
  MCP or by a play acting for someone adds that person. Archiving, locking, and moving a doc are not edits and add
  nobody.
- Anyone who can open a doc may watch it or stop watching it, which takes `docs:read` checked where the doc lives
  (ADR 0087), and only for themselves: nobody adds or removes someone else.
- Stopping sticks. The `doc_watchers` row stays with `watching` off, so the person's own later edits do not add them
  again; watching again turns it back on.
- `doc.updated` notifies the watchers, minus the editor and minus anyone who can no longer open the doc. `doc.created`
  does the same, so it notifies nobody: the creator is the only watcher and the actor. A mention still notifies the
  person mentioned whether or not they watch, and does not make them a watcher.
- Migration 0050 makes every existing doc's creator and each distinct named-version author a watcher, skipping anyone
  who is not a user. An upgraded instance therefore stops notifying members who never touched a doc.
- Watching and stopping publish `doc.watchers.changed`, scoped to the doc's project and pushed live to whoever can read
  the doc. Being added by a save publishes no event of its own; `doc.created` and `doc.updated` already name the actor.
- Over HTTP the routes are `GET /api/docs/{id}/watchers` and `PUT` and `DELETE /api/docs/{id}/watchers/me`. Over MCP,
  `doc_get` returns the watchers and whether the caller is one, and `doc_update` takes `watch`, so the tool count is
  unchanged.

The trade-offs: someone who wants every doc in a workspace has to watch each one, since there is no workspace-wide or
folder-wide watch. A plain save adds its editor even for a one-word fix, the same as an issue tracker; the opt-out is
how they leave. Unnamed versions recorded no author, so the backfill can only find creators and named-version authors,
and a past editor who was neither is not a watcher after the upgrade.

Rejected: letting someone add another person as a watcher, which turns watching into assigning; making a mention add
its subject as a watcher, which would make one mention a permanent subscription; a separate `doc_watch` tool, which the
tool ceiling has no room for when `doc_update` already carries the doc's other toggles.

Decided 2026-10-01.
