# 10: Build matching, the run queue, limits, and pause

Type: task
Status: open
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
