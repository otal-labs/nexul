# Wayfinder map: ticket flow

Charted 2026-10-02 with the owner. Two tracks that keep a ticket page
readable as agents work on it: the thread moves beside the body, and an
agent adds context as a note in the thread instead of growing the body.

## Destination

Every decision locked for both tracks, ready to slice into build tickets:
the thread as a left column on the ticket page, and notes (an Agent message
in the ticket's thread carrying a markdown file attached to the ticket,
opened, edited, and saved in a dialog). Planning only; the build is its own
step after this map.

## Notes

- Decided while charting: one map for both tracks; the thread sits in a left
  column beside the body on wide screens, the way a doc's "On this page" list
  does, and drops back under the body when the screen is narrower; a note's
  file is `.md` and belongs to the ticket, so it also lists under the
  ticket's attachments; the body is the ticket's spec and changes only when a
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
