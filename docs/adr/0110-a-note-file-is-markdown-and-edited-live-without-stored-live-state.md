# A note's file is markdown, edited live with its live state kept only in memory

A note's file (ADR 0108) is edited live the way a doc body is: people in the same note see each other's typing.
The doc machinery stores two things, the rich-text body and the room's Y.js update log beside it (ADR 0026, ADR
0109), and that log's tables reference `docs(id)`. A note has neither a rich-text body nor a doc row.

Decision: the markdown file is the only thing stored, an exception to ADR 0026. The relay, protocol, and browser
provider are the docs ones, served by a second collab hub at `/ws/collab/notes/{id}` keyed on the note's message id.

- The hub takes the actions its rooms join under instead of hardcoding the docs ones. A note's room takes
  `tickets:write` on the note's ticket, through message, conversation, and ticket, in edit and view mode alike, so a
  reader never joins and sees the static render. A deleted note refuses joins and drops commits.
- The room's live state lives in memory (`collab.MemoryStore`): the newest snapshot and the increments after it. The
  first editor to join an empty room seeds it from the file. The state is dropped on restart and whenever the file
  is replaced from outside.
- A commit converts the editor's tree with `richtext.ToMarkdown`, the title ignored, and overwrites the file's bytes
  and moves the message's `updated_at` in one transaction with `chat.message.updated`. The session's dedupe skips an
  unchanged body, and merely opening a note writes nothing, as with docs.
- An agent replaces a note's file through `message_post` with `file.note_id`; browsers and integrations through
  `PUT /api/chat/messages/{id}/note`. Both run the chat use-case under `tickets:write`, which writes through the
  room's reset (ADR 0109), so stale commits are dropped and every editor reloads from the new file.

The trade-off: markdown the editor cannot express (raw HTML, front matter, reference links) is flattened on the first
person's save, and a restart, or a replace from outside, discards the last few seconds of typing in an open room. In
exchange there is no second store to keep in step with the file, and an agent reading or writing a note sees exactly
what people see. A browser that reconnects after a restart joins an empty room with its old state, the same gap ADR
0109 names for a lost reset seq.

Amends ADR 0050: the note dialog edits live in place like a doc page, but inside a dialog, because a note belongs to
a thread rather than a page of its own.

Decided 2026-10-02.
