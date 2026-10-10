# 15: Walk auto plays through live

Type: task
Status: open
Blocked by: 11, 12, 13, 14

## Question

On a real install with a paired harness: compose "When a ticket becomes
unblocked, if type is Bug, High" on Fix with AI, unblock a bug and watch it
queue and run; take the computer offline and see it wait; re-block a
queued ticket and see it skipped; drive a test-fail loop into the daily cap
and resume; switch on the decisions check; call `runPlay` from an
automation. Fix what breaks in follow-up PRs, then settle the phone
question in the map's "Not yet specified".

## Notes
- A person with `plays:read` + `autoplays:read` but no `plays:write`
  can't open the play dialog, so never sees the Auto plays tab; decide in
  the walkthrough whether a read-only way into a play is needed.
