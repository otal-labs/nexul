# 02: Where a note's file lives and how its message points at it

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

A note's `.md` file is an attachment owned by the ticket (decided while
charting), and the note's message sits in the ticket's thread. How does the
message point at the file, and what happens when either side goes away?

- The link: a markdown line in the message body, the way chat images are
  referenced today, or a field on the message.
- Deleting the file from the attachments rail: the message keeps a "file
  removed" pill, or the message goes too.
- Deleting the message: the file stays on the ticket, or goes with it.
- The events a note publishes (catalog rows, outbox writes) and the live
  push that makes the pill appear without a refresh.
- Who can open the file: anyone who can read the ticket, the same as other
  ticket attachments.

Technical: arrives as a decided answer for a yes or no.

## Answer

The owner moved the file from the ticket to the thread.

- The file is an attachment owned by the ticket's conversation, linked
  from the note's message through the existing `messages.attachment_id`
  field (in the schema and the message payload, unused until now), not a
  markdown line. The message body is the agent's one-line summary. Agent
  messages render through a path that never parses chat image lines, and a
  field makes "this is a note" a fact rather than parsed text.
- The message and its file are one thing: deleting the message deletes the
  file. The file never lists under the ticket's Attachments rail; the pill
  in the thread is the only way in.
- Anyone who can read the ticket can open it, the same access its thread
  already has.
- No new event topic: `chat.message.created` already carries the whole
  message and writes its outbox row; its catalog entry gains a description
  of `attachment_id`. The live message upsert in `useLiveEvents.tsx` makes
  the pill appear without a refresh.
- At build time, record an ADR: a note is an Agent message carrying a
  conversation-owned markdown file, amending ADR 0027's "attachments have
  no MCP tools" (a note carries text, not bytes).
