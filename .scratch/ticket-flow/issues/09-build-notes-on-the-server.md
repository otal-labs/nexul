# 09 — Notes on the server: post, read, delete

**What to build:** A note as decided in [Where a note's file lives](02-note-file-and-message-link.md)
and [How an agent leaves a note](03-agent-leaves-a-note-over-mcp.md).
`message_post` and the gateway's post-message route take an optional file
(name and markdown) accepted only on a ticket's thread; the file is a
conversation-owned `.md` attachment linked through `messages.attachment_id`,
and the message is authored by the Agent on behalf of the caller, so no
turn starts. Posting a note needs `tickets:write`. `message_list` returns a
note's markdown with its message. Deleting a note's message needs
`tickets:write`, deletes its file and the images its markdown references
in the same transaction, and the generic attachment delete refuses note
files. The file route keeps the forever cache for inline images only.
An agent turn on a ticket carries note text: non-image note files no longer
become "[attachment omitted]", and follow-up prompts include notes.

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

- [ ] A note posted over MCP and over HTTP shows as "Agent · via <person>", starts no turn, and its file never lists under the ticket's attachments
- [ ] Posting a file anywhere but a ticket's thread, or without `tickets:write`, is refused in the use-case
- [ ] `message_list` and the agent turn prompt both carry the note's markdown
- [ ] Deleting the message removes the file and its referenced images; deleting the file alone is refused
- [ ] `chat.message.created`'s catalog entry describes `attachment_id`; the web upserts the note live and refreshes nothing else
- [ ] The existing system "notes" in chat (`PostSystemMessage`, `byNoteBody`, `passedNote`) are renamed so "note" means only this
- [ ] `clearStream` in `useLiveEvents.tsx` and `answeredAfter` in `MessageList.tsx` skip notes, so a mid-turn note neither ends the working bubble nor answers an open question
- [ ] An ADR records that a note is an Agent message carrying a conversation-owned markdown file, amending ADR 0027's "no MCP tools for attachments"; Access.tsx lists the note permission
- [ ] The MCP surface test stays under its budget with no new tool
