# 12 — The note pill and its dialog

**What to build:** The look locked in [The note pill and its file dialog](05-note-pill-and-dialog-look.md),
ported from variant A on `proto/thread-column` (commit 67dcb174) without
its switcher or fake presence: the compact file pill under an Agent note's
one-liner, opening a centred dialog the width of the ticket body card with
the file name, the summary title, and the doc presence bar, and the file
in `RichTextEditor` joined to the note's live room. Always editable for
anyone who may write the ticket; a reader gets the read-only render.
Images paste into the note as files of the same thread. Deleting a note
is offered to ticket writers.

**Blocked by:** 11

**Status:** ready-for-agent

- [ ] The pill shows in the thread pane and in Chat for the same thread; clicking it opens the dialog
- [ ] Editing is live with presence, saves on the docs cadence, and closing the dialog flushes
- [ ] A pasted image shows in the note and survives a reload
- [ ] A reader sees the render with no editor and no delete
- [ ] Checked at 768, 1024, and 1440px; Frontend Commandments self-review done
