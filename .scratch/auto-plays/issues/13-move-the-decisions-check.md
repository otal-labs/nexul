# 13: Make the decisions check an ordinary play with a seeded auto play

Type: task
Status: resolved
Blocked by: 07, 10

## Question

The decisions check becomes a play in every workspace with one seeded auto
play, "Ticket enters stage done", run on whoever caused it, off by default.
A migration carries each workspace's current on/off state onto the seeded
auto play. The switch leaves the Automations page, the special
`decisions-check` id on `PATCH /api/workspaces/{id}/plays/decisions-check`
and in `play_update` goes, and the hardcoded consumer in
`internal/plays/decisions_check.go` is replaced by the auto play path. The
existing "Decisions check didn't run" retry keeps working. Update the user
guide's decisions check and automations pages.

## Notes

- Rewrite `CONTEXT.md`'s Decisions check entry (an ordinary play with a
  seeded auto play, switched on the play's page) and the Play entry's
  "fires with one button" to allow auto plays.
- Seed the play with a label free in each workspace (play labels become
  unique per workspace in 14; if "Decisions check" is taken, add " (2)").
- Move the check onto 10's queue: its consumer goes; the seeded auto play
  is `ticket.entered_stage` with stage done, run on `causer`.

## Answer

Built in PR #540: the decisions check is a seeded ticket play (built-in
key `decisions-check`, shown in done) with a seeded auto play (ticket
enters done, run on whoever caused it), migration 0087 carrying each
workspace's old switch onto it and pointing old trails at the play. The
hardcoded consumer, the `.../plays/decisions-check` routes and the special
`play_update` id are gone; the "didn't run" notice and its retry remain.
A queue row's id now derives from the event and the auto play, so a
redelivered event never queues a second run.
