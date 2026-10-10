# 06: How a ticket shows queued, skipped, didn't run, and paused

Type: prototype
Status: resolved
Blocked by: None — can start immediately

## Question

What a ticket page (and its card on the board, if at all) shows for an
auto play:

- **Queued**: a chip naming the play and who it waits on ("Fix with AI
  queued for alice, computer offline"), with a way to cancel it.
- **Skipped**: a muted line in the thread ("Fix with AI skipped: no longer
  unblocked").
- **Didn't run**: nobody to run on, or that person is excluded from the
  play, with a way to run it by hand. The decisions check's existing
  "didn't run" is the precedent.
- **Paused**: "Auto plays paused on this ticket" after the daily cap, with
  a resume button.

Prototype with `design-mode` against the ticket page as it is; the owner
picks by looking.

## Answer

Picked in auto design mode (2026-10-10), runner-up reported to the owner
for a swap. Prototype: branch `proto/auto-plays`, variant A.

- **Current state lives in the rail's Plays section**, which shows whenever
  something is queued or paused even if no play applies to the stage.
  Queued: an `info` clock icon, "Fix with AI queued" over a muted
  "<person> · computer offline" (or "no free slot", "ticket busy"), a ghost
  Cancel on the right. Paused: a `warning` pause icon, "Auto plays paused"
  over "5 runs today", a ghost Resume.
- **What happened lives in the Thread** as one-line event lines: a small
  icon in a circle, the play name in medium weight, the rest muted, a mono
  time. Skipped: a muted icon, "Fix with AI skipped: no longer unblocked".
  Didn't run: a `warning` icon, "Fix with AI didn't run: nobody is the
  ticket's developer", and an outline "Run it" for the viewer.
- Status is the icon's hue beside plain text, never a tinted chip. No
  marker on the board card: it would compete with the status dots and go
  stale.
- Below 1440px the rail sits under the body, so queued and paused are below
  the fold there, like every other rail section.
- Rejected: notice strips above the body (four stacked push the body down
  about 300px), and everything in the Thread (current state scrolls away
  with the history and wraps to five lines in the narrow pane).
