# 05: The note pill and its file dialog

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

In the thread, a note is one line from the Agent plus a file pill. Clicking
the pill opens a dialog that renders the file the way the ticket body
renders, and lets anyone who may write the ticket edit it live, the way a
doc is edited. What do the pill and the
dialog look like?

- The pill inside an Agent message in the thread pane.
- The dialog: reading view, the switch into editing (or always editable),
  who else is in it, and closing.

Prototype the candidates live; the owner picks by looking.

## Answer

Variant A on the `proto/thread-column` branch (commit 67dcb174), picked by
the owner over a preview card with a side sheet and a linked title with a
near-fullscreen dialog.

- In the thread, the note is the Agent's one-line summary with a compact
  file pill under it, the same shape as the existing attachment pill:
  file icon, name, size.
- Clicking the pill opens a centred dialog the width of the ticket body
  card. Its header carries the file name in mono, the summary as the
  title, and the doc page's presence bar ("Live", who is in, "updated …").
  Below it the file renders in the same card as the ticket body.
- Always editable in place for anyone who may write the ticket, with no
  Edit switch, the way a doc page is its own editor. A reader gets the
  same dialog with the read-only render.
