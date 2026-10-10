# 12: Build the queued, skipped, didn't-run, and paused signals

Type: task
Status: open
Blocked by: 06, 10

## Question

Build what 06 locked on the ticket page (and board card if chosen), wired
to 10's records and live push. Web tests; verified at 768, 1024, and
1440px.

## Notes
- Read the queue through 10's routes and the `play.queued` /
  `play.queue_updated` / `play.queue_resumed` follower (`TrailHooks.tsx`,
  `web/src/models/PlayQueue.tsx`).
- Add the editor's "Right now: N queued · 1 waiting on <person> (computer
  offline)" line under the auto plays list (05's Answer), from the same
  queue data scoped to the play.
- When the rolling day frees a paused ticket with nothing waiting, no event
  fires, so an open page keeps "paused" until it refetches: publish one
  from the dispatcher's timer, or derive "paused" from the oldest counted
  run's time on the client.
