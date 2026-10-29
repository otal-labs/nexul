# When editing a doc writes a version

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

Typing in the doc editor saves through collaborative-editing flushes, and a
flush writes no version, so a doc's history today only has API and MCP saves
and named snapshots. What should write a version from the editor: the room
going idle for some minutes, the last editor leaving, one per editing
session per person, or something else? And should a save that changes
nothing stop writing a row, here and for memories?

## Answer

Decided while working [What a ticket's history is](02-what-a-ticket-history-is.md),
because one rule covers both doc and ticket descriptions.

- The live editor writes one version per sitting. The version is written
  once the doc or ticket description has had no edits for about 5 minutes,
  or when its last editor leaves. It is credited to everyone who edited
  during the sitting.
- Live updates to other viewers and the editor's background saves are
  unchanged. Only the history entry waits for the sitting to end.
- A save that changes nothing writes no version, whether it comes from the
  editor, the API, MCP, or a memory save.
- Why: Notion versions every 10 minutes while editing and again after you
  stop; Confluence's live pages version 10 minutes after a session ends;
  Figma checkpoints every 30 minutes; Tiptap's hosted collaboration
  versions on a timer. None of them versions every change. One version per
  sitting keeps the 10 kept versions covering the last 10 sittings, where
  a timer would fill them within one long session.
