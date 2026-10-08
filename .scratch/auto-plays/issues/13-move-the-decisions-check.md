# 13: Make the decisions check an ordinary play with a seeded auto play

Type: task
Status: open
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
