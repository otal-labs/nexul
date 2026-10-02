# 15 — Walkthrough on a real instance

**What to build:** An end-to-end run on nexul-box after 08 to 14 land: a
paired agent leaves a note on a ticket over MCP, two browsers edit it live,
the agent replaces it mid-edit, a reader opens it read-only, search finds
its text, the phone opens it, and deleting the message removes the file.
Findings become fix-up tickets here.

**Blocked by:** 08, 10, 12, 13, 14

**Status:** ready-for-agent

- [ ] Every step above passes or has a fix-up ticket

## Findings

Run on a native install of `3d1dd37a` in an Incus box (2026-10-02). Steps one
to nine passed: thread pane layout and resize, an agent note over MCP, two
writers live in the note, the agent replacing it mid-edit, the reader's live
read-only view, note search on the web and over MCP, the table guard, a doc
written over MCP while open, and deleting a note. Open:

- The phone step is untested; it needs the owner's device.
- A note with a markdown table opens read-only with the guard line and its
  file stays byte-identical, but the read-only render shows no table at all:
  the heading and paragraphs around it render, the rows are dropped. Readers
  see the same. Recheck once the editor supports tables.
- A doc or note whose last block is an image opens with the editor's
  selection on that image. Tabbing into the editor and typing replaces the
  image with the typed text, and the next commit writes that loss to the
  file; typing straight after a click can do the same before the click's
  selection lands. Repro: post a note ending in `![x](https://example.com/x.png)`,
  open it, press Tab once, type, wait for the save. Same on a doc, so it is
  the shared editor, not notes. Needs a choice between moving the selection
  off the image after a seed, sync, or reset, and keeping a trailing
  paragraph after a block image (which changes the stored markdown).
- Every fresh page load asks `GET /api/projects?workspace_id=` before a
  workspace is selected and gets a 400. Gating the query on a workspace id
  fixes it, but about twenty component and page tests render without a
  selected workspace and would need one first.
