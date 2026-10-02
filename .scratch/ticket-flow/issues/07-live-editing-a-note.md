# 07: Live editing a note

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

A note's markdown file is edited live, the way a doc body is: people in the
same note see each other's changes as they type, and a save replaces the
file with no history. How does the doc collaboration machinery serve a
note's file?

- What the live session is keyed on and who may join it (anyone who may
  write the ticket).
- What is stored: the markdown file as the source of truth with the live
  state rebuilt from it, or a live state persisted beside it.
- When the file is written back, and what the thread and other viewers see
  when it changes.
- How an agent's edit over MCP lands while people are in the session.
- Whether the dialog reuses the doc editor as is.

Technical: arrives as a decided answer for a yes or no.

## Answer

The docs relay, browser provider, and editor carry over; docs' persisted
live state does not (its tables reference `docs(id)`).

- A second collab hub at `/ws/collab/notes/{id}`, keyed on the note's
  message id, with the same protocol and provider. The hub takes its edit
  and view actions instead of hardcoding the docs ones; joining checks
  `tickets:write` through message, conversation, and ticket. Readers see
  the static render and never join.
- The markdown file is the only thing stored. The live state lives in
  memory: the first joiner of an empty room seeds it from the file, and it
  is dropped on restart and when the file is replaced from outside.
- The file is written back on the docs cadence (a commit every five
  seconds while editing, a flush on close, unchanged bodies skipped),
  converted with `richtext.ToMarkdown`, replacing the bytes and publishing
  `chat.message.updated` in the same transaction. The file route stops
  caching non-image files forever.
- An agent's edit replaces the file, then resets the room: it drops the
  memory, rejects in-flight commits from before the reset, and tells every
  editor to reload from the file. Keystrokes from the last few seconds are
  lost, matching "a save replaces the file". Agents edit a note through
  `message_post` (an existing note's id plus new content), browsers and
  integrations through a matching HTTP route.
- Only people who may write the ticket edit or delete a note.
- The dialog reuses `RichTextEditor` with its collab prop. Images can be
  pasted into a note (owner's call): they upload as files of the same
  thread, and deleting the note deletes the images its markdown references.
- Markdown that the editor cannot express (raw HTML, front matter,
  reference links) is flattened on the first person's save.
- At build time, record an ADR: a note's file is markdown-canonical and
  edited live without stored live state, an exception to ADR 0026, with
  ADR 0050 amended for the note dialog.
- Found while answering, fixed separately: a server-side doc body write is
  lost once the doc has had a live session, two people opening a cold doc
  can both seed it, and anyone who can read a conversation can delete files
  in it.
