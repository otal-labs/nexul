# 05 — What a play run sends to the harness

**Type:** grilling
**Status:** resolved
**Blocked by:** 01, 02

## Question

A chat turn today sends: fixed Agent instructions, the memories index, the
ticket (title and body) for a ticket thread, the unsynced conversation, and
the request. A play run adds base instructions, inlined memories, and custom
instructions, and its request is not a human message.

1. Block order and wording: where the play's base instructions go relative
   to the fixed Agent instructions, how the inlined memories are framed, and
   how the custom instructions are framed so "steer it in a different
   direction" wins over the base text when they conflict.
2. The request block: does the run post a visible message in the thread
   ("Onik started Fix with AI") that doubles as the prompt's request, or is
   the request synthetic and the thread only shows a system note?
3. Session reuse: the ticket thread's harness session is reused across runs
   and chat mentions, so the incremental prompt path applies. Confirm a play
   run uses it too rather than always opening a fresh session.
4. Tools the two seeded plays rely on and which exist: `ticket_create`,
   `ticket_update`, doc
   reading. "To tickets via AI" needs to know the doc's project and the
   board's columns: are `project_get` and a project lookup enough?
5. Draft the seeded instructions for "Fix with AI" and "To tickets via AI"
   and put them to the owner, including what "Fix with AI" says about
   merging when a selected memory (the owner's "merge-this" idea) permits
   it; the ticket then lands on Done through the existing finish rule and
   the play's move-to skips, per ticket 03.
6. A memory body is rich text exported to markdown: what the export does
   with an embedded image or attachment link (drop it, keep the link, note
   that it was omitted), since the harness cannot fetch Nexul attachments.
7. Size: selected memories inlined in full plus the ticket plus history must
   fit the harness limit. Which block is trimmed first when it does not?

## Answer

Resolved 2026-09-16 with the owner (one grilling round).

- **Block order**: the fixed Agent instructions (identity, whoami check,
  memories index) as today; the target (ticket or doc) as markdown; the
  conversation so far; then the run request. The request holds the play's
  name and base instructions, the inlined memories under "Memories the user
  selected for this run, follow them", and the custom instructions under
  "Instructions from <user> for this run; where these conflict with the
  play's instructions, these win". Everything play-specific rides inside
  the request so the reused-session incremental prompt carries it too.
- **Mandatory memories.** The owner's answer to "where do the standing
  rules live": a memory can be marked **always included**. Such memories
  are inlined in every play run in that project, shown ticked and locked in
  the dialog, and count toward the run ceiling first. The fixed Agent block
  stays minimal (identity, whoami, how memories and tools work); the
  editable "main stuff" for a project is a mandatory memory the owner
  writes, and the seeded defaults include one per project titled "Working
  in this project" with a starter body. Whether chat turns also inline
  mandatory memories is left to the spec writer's judgment; the
  recommendation is yes, same rule, so chat and plays never disagree.
- **A visible message is the request**: the click posts a real message
  from the starter, "Started <play>" plus their custom instructions, and
  the Agent's reply lands under it.
- **Trim order** when over the turn limit: conversation history first, then
  the target body with a note; never the inlined memories or the custom
  instructions.
- **Images in an inlined memory**: the owner asked why the harness cannot
  receive them as it does from a phone. That is a fact to establish, ticket
  11. Until it is, the export writes "[image omitted: <alt>]" and keeps
  links; if the harness accepts attachments, images go along as
  attachments instead.
- **Seeded instruction texts** ship as drafted in the question (Q5 and Q6),
  editable by the owner like any play.

### Addendum, after tickets 11 and 12

T3 Code accepts images on a turn as base64 data URLs, up to 10 MiB each,
and never fetches a URL. So an inlined memory's images **go along as
attachments**: the server reads the attachment bytes it already stores
(ADR 0027) and hands them to the harness through a harness-neutral
`Attachments []Attachment{Name, MIME, Bytes}` on the turn prompts; the T3
client encodes them. The markdown keeps the image's name in place so the
text and the attachment line up. Images over the per-attachment limit, and
non-image attachments, fall back to "[attachment omitted: <name>]". The
same rule applies to images in a ticket or doc body carried as the target.
