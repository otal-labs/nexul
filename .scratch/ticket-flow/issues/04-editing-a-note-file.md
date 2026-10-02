# 04: Editing a note's file

Type: grilling
Status: resolved
Blocked by: 02

## Question

Attachments are fixed bytes today. A note's file can be edited and saved
from its dialog. What does a save do?

- Overwrite the bytes in place, or keep versions with their author the way
  memories do, with a way back to an earlier one.
- Who may edit: anyone who can write the ticket, or only the person the
  agent posted for.
- Whether an agent can edit a note's file over MCP too, so agents stay
  peers of the browser.
- What other viewers see when a file they have open is saved (live push,
  a stale warning, or nothing).
- Whether only note files are editable, or every text attachment.

## Answer

- Live, the way a doc is edited: people editing the same note see each
  other's changes as they type. How the doc collaboration machinery serves
  a note's file is [Live editing a note](07-live-editing-a-note.md).
- A save replaces the file. No version history, the same as the ticket
  body beside it.
- Anyone who may write the ticket edits its notes, and agents can edit
  them over MCP too.
- Only note files are editable; every other attachment stays as uploaded.
- At build time, record an ADR: a note's file changes in place, so
  attachment bytes are no longer immutable, and the file route stops
  telling browsers to cache note files forever.
