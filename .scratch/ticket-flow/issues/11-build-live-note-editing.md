# 11 — Live editing a note's file on the server

**What to build:** As decided in [Live editing a note](07-live-editing-a-note.md):
a second collab hub at `/ws/collab/notes/{id}` keyed on the note's message
id, the hub taking its edit and view actions instead of the hardcoded docs
ones, joining gated on `tickets:write` through message, conversation, and
ticket. An in-memory store holds the live state; the first joiner of an
empty room seeds it from the file. The notes writer converts the editor
tree with `richtext.ToMarkdown`, replaces the attachment bytes, and
publishes `chat.message.updated` in one transaction, refusing writes once
the message is deleted. An agent replaces a note's file through
`message_post` (the note's id plus new content) and browsers through
`PUT /api/chat/messages/{id}/note`; either resets the live room with the
reset frame the docs fix introduced, so stale in-flight commits are
dropped and every editor reloads.

**Blocked by:** 09, and the docs live-session fix (its reset frame and
single seeder) merged to master

**Status:** ready-for-agent

- [ ] Two people editing one note see each other's changes; the file on disk matches within one commit interval
- [ ] An agent's replace while people are in the room wins; nobody's next commit overwrites it
- [ ] Opening a note without typing writes nothing
- [ ] A reader without `tickets:write` cannot join the room or replace the file
- [ ] An ADR records that a note's file is markdown-canonical and edited live without stored live state (an exception to ADR 0026), and amends ADR 0050 for the note dialog
