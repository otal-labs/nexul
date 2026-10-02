# A note is an Agent message carrying a markdown file of its conversation

Amends ADR 0027: attachments still have no tools of their own, but `message_post` now writes one kind of
conversation file, a note's.

An agent working a ticket kept growing the ticket's body with context nobody asked for, so the body stopped reading
as the spec. A note gives that context a place of its own in the ticket's thread.

Decision: a note is one message in a ticket's thread, authored by the Agent on behalf of the person whose token
posted it (`author_kind` agent, `author_id` the caller), with its markdown file stored as an attachment owned by that
conversation and linked through `messages.attachment_id`. The link is the field, never a markdown line in the body,
so "this is a note" is a fact rather than parsed text.

- `message_post` and `POST /api/chat/conversations/{id}/messages` take an optional `file` (name and markdown). Only a
  ticket's thread takes one, the name always ends in `.md`, and posting it needs `tickets:write` on the ticket,
  checked in the chat use-case so both adapters agree. The file and the message are written in one transaction.
- A note starts no turn, because turns start only for a person's own message mentioning `@Agent`. A post without a
  file still posts as the caller.
- `message_list` returns a note's markdown with its message, and an `@Agent` turn on the thread reads it as text in
  its history, follow-up prompts included, rather than as an omitted attachment.
- The message and its file are one thing. Deleting the message needs `tickets:write`, not authorship, and deletes the
  file and the images of the same conversation its markdown points at, in the same transaction. The attachments
  delete refuses a note's file, and the file never lists under the ticket's own attachments.
- No new event: `chat.message.created` already carries the whole message, and its catalog entry describes
  `attachment_id`.
- The file route keeps telling browsers to cache inline images forever and every other file `no-cache`, because a
  note's file is edited in place.

The trade-off: a note's file goes through chat's repository rather than the attachments use-case, so its content
type is fixed to markdown instead of sniffed, and chat repeats the 10 MB cap. Accepted for one transaction instead of
an upload, a post, and a clean-up when the post fails.

Rejected: a new tool, which the tool budget had no room for; carrying the file as bytes through a general upload
tool, which ADR 0027 already declined; and a markdown link in the body, which agent messages never parse.

Decided 2026-10-02.
