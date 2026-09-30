# A locked doc is a guard, not a permission

A finished doc (a signed-off spec, a runbook) stays open in the editor for anyone who can write it, and one stray
keystroke in a live session is saved within seconds. Docs needed a way to say "this one is done, leave it".

Decision: a doc carries a `locked` flag. While it is set, the doc's title and body refuse every change: the use-case
answers an edit over HTTP or MCP with a conflict, the collaborative relay drops every update and commit for it
without storing or relaying them, and the browser opens no edit session, so the page reads the doc instead of editing
it. Anyone holding `docs:write` on the doc locks or unlocks it (`POST /api/docs/{id}/lock` and `/unlock`, `locked` on
`doc_update`); there is no permission bit of its own. Archiving, cloning, and deleting still work on a locked doc, and
a clone starts unlocked. The change travels as `doc.updated` with `locked` in the doc, so every open page flips
without a refresh.

The trade-off: a lock stops accidents, not people. Whoever could edit the doc can unlock it and edit, which is what
keeps a lock from being a one-way door when its author is away. A doc that must stay unchanged against someone who
means to change it takes that person's `docs:write` away through a permission overwrite (ADR 0042), which is the tool
for that job.

Rejected: a `docs:lock` bit, which would make a lock that only some people can lift and a second answer to "who may
edit this"; and refusing the relay's edit joins to a locked doc instead of its writes, which would not stop a session
that was already open when the lock landed.

Also: the relay now refuses writes from a participant who joined in view mode, which it had been storing and
relaying although the join only checked `docs:read`.

Decided 2026-09-30.
