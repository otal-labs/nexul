# 10: Build matching, the run queue, limits, and pause

Type: task
Status: resolved
Blocked by: 03, 08, 09

## Question

Build what 03 decided: consumers for the six moments that match enabled
auto plays and queue runs on the right person, the queue with priority,
per-person and per-ticket slots, the re-check at the front, skipped and
didn't-run records, per-auto-play limits, the per-ticket daily cap and
resume, and the wake-ups (run ended, computer online, resume). HTTP and
MCP to read and cancel queued runs and resume a paused ticket. Go tests
for each path, integration tests against real SQLite.

## Notes

- Auto runs skip `checkPlay`'s show-when stage gate (03's answer); manual
  presses keep it.
- `doc.settled` carries `first`: `first: true` matches the stored moment
  `doc.created`, `first: false` matches `doc.changed`. `ticket.unblocked`
  matches `ticket.unblocked`; the other ticket moments map from the existing
  topics (`ticket.created`, `ticket.status_changed` into a stage,
  `ticket.developer_changed` and `ticket.tester_changed` to someone,
  `ticket.test_failed`).
- The matcher reads `idx_auto_plays_moment` (workspace, moment, enabled).

## Answer

Built in PR #530: one matching consumer for the eight moments, the
`play_queue` and `play_queue_resumes` tables (migration 0084), and
`Runner.RunQueue`, which wakes on kicks, `play.run_finished`, boot, and a
timer to the next held row (an offline person's one-minute retry, or a
paused target's oldest counted run leaving the day). Auto runs skip the
show-when stage gate; an offline computer queues and never writes a failed
trail; "nobody to run on" and "may not run the play" are decided at match
time as didn't-run with a failed trail. The cap counts started runs over
24 hours from the later of a day ago and the last resume, for tickets and
docs alike. Surfaces: `GET /api/plays/queue`, `POST /api/plays/queue/resume`,
`POST /api/plays/queue/{itemID}/cancel`;
MCP `trail_list` `queue`, `trail_update` `cancel`, `play_run`
`resume_auto_plays`; events `play.queued`, `play.queue_updated`,
`play.queue_resumed` with live push and a web follower. Cancel: the person
the run lands on or `autoplays:write`; resume: the ticket's developer or
`autoplays:write`, only while paused. `ticket.developer_changed` and
`ticket.tester_changed` gained `actor`.
