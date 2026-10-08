# 06: How a ticket shows queued, skipped, didn't run, and paused

Type: prototype
Status: open
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
