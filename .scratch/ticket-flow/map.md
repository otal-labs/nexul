# Wayfinder map: ticket flow

Charted 2026-10-02 with the owner. Two tracks that keep a ticket page
readable as agents work on it: the thread moves beside the body, and an
agent adds context as a note in the thread instead of growing the body.

## Destination

Every decision locked for both tracks, ready to slice into build tickets:
the thread as a resizable left pane on the ticket page, and notes (an Agent
message in the ticket's thread carrying a markdown file, opened and edited
live in a dialog). Planning only; the build is its own
step after this map.

## Notes

- Decided while charting: one map for both tracks; the thread sits in a left
  column beside the body on wide screens, the way a doc's "On this page" list
  does, and drops back under the body when the screen is narrower; a note's
  file is `.md`; the body is the ticket's spec and changes only when a
  person asks, everything an agent adds afterwards is a note, and an agent
  may suggest the person edit the body when the spec itself looks wrong.
- Technical tickets arrive as a decided answer for a yes or no; look tickets
  are prototypes the owner picks from by looking. Plain language with the
  owner, no ticket numbers or section codes in questions.
- Web screens are judged at 768, 1024, and 1440px.
- Prototype tickets: `/prototype` inside `design-mode`. Grilling tickets:
  `/grilling` + `/domain-modeling`.
- Reference: ticket SRC-3 on the owner's instance keeps its "Prerequisites
  carried forward from SRC-2" body section on purpose, as the example of
  what a note replaces.
- Grounding: `CONTEXT.md` (Note, Agent, Doc thread, Memory, Trail), ADR 0027
  (attachment bytes in SQLite), ADR 0060 (chat is a page). Code:
  `web/src/components/ticket/TicketPageBody.tsx` (the two-column grid),
  `web/src/components/doc/DocDetail.tsx` (the left contents list),
  `web/src/components/chat/TicketThreadSection.tsx` (the fixed-height
  thread), `web/src/components/attachment/` (pills, attachments rail),
  `internal/chat/mcp.go` (`message_post`, posts as the person, a turn
  starts only on an @Agent mention), `internal/chat/usecase.go`
  (`PostAgentMessage`), `internal/attachments/model.go` (an attachment has
  exactly one owner).
- Related, read and do not act on: `.scratch/ticket-workflow-depth/`
  (comments and activity on tickets, needs-triage).

## Decisions so far

- [The thread column on the ticket page](issues/01-thread-column-look.md):
  full width, the thread pane flush to the sidebar with a drag-resizable
  width and the composer at its foot; the rail flush right.
- [Where a note's file lives and how its message points at it](issues/02-note-file-and-message-link.md):
  the file belongs to the thread, linked by the message's attachment field;
  message and file are deleted together and never list under the ticket's
  attachments.
- [How an agent leaves a note over MCP](issues/03-agent-leaves-a-note-over-mcp.md):
  a file option on `message_post`, posted as the Agent and never starting a
  turn, needing `tickets:write`; agents read notes through `message_list`.
- [Editing a note's file](issues/04-editing-a-note-file.md): live like a
  doc, by anyone who may write the ticket, agents included; a save replaces
  the file with no history; only notes are editable.
- [What agents are told about the body and notes](issues/06-what-agents-are-told.md):
  the tool descriptions and one line in every agent turn on a ticket.

## Not yet specified

- Whether a new note reaches anyone: an Inbox entry, a notification to the
  ticket's people, or nothing beyond the thread.
- Whether a note's file is found by search, alongside the ticket body.
- Whether a play run leaves its closing summary as a note instead of a plain
  message.
- How the phone app shows a note's pill and file.

## Out of scope

- Notes on docs. A doc's thread lives on the chat page, not beside the doc;
  if it is ever wanted, it is a new map.
- Moving SRC-3's carried-forward section into a note. A one-off, kept as the
  reference example.
