# 15: Walk auto plays through live

Type: task
Status: open
Blocked by: 11, 12, 13, 14, 16

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

## Comments

- 2026-10-10: the harness-free part was walked on a copy of the demo
  instance; its defects were fixed in #545 and #546, and a switched-off
  automation's token still working is in `.scratch/latent-bugs/`.
- 2026-10-11: the owner chose to walk the real runs on the live instance
  with their paired computer, using a test play with one-line instructions;
  the phone question is settled in the map (no signals for now).
