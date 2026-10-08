# 03: The run queue: slots, online computers, re-checks, and the daily cap

Type: research
Status: open
Blocked by: None — can start immediately

## Question

A matching moment does not start a run; it queues one for the person the
auto play runs on. One queue per person, High before Normal before Low,
oldest first within a level. A run starts when the person has a free slot
and their computer is online, after its conditions are checked again.

- How a play run starts today (`internal/plays/run.go`, the decisions
  check consumer), how the server knows a run ended (the trail), and what
  "computer online" means to the server (pairing, tunnel, presence).
- Where the queue lives (a table, so a restart loses nothing; ADR 0119
  reattach is the precedent) and what wakes it: a run ending, a computer
  coming online, a new item queued, a resume pressed.
- The default concurrency per person, and one auto run per ticket at a
  time. Do manual presses take a slot, and do they ever wait behind auto
  runs? (Recommended: manual presses never queue.)
- The per-ticket daily cap across all auto plays (default 5): counting,
  the "Auto plays paused on this ticket" state, and what resume clears.
- "Didn't run" (nobody to run on, or the person is excluded from the
  play) versus "skipped" (no longer matches at the front): what each
  stores and how the ticket reads it.
- Loops: a run's own changes cause moments (Fix with AI moves the card to
  review). Is the daily cap enough, or does a run's own change need
  marking?

Findings go in `research/03-the-run-queue.md`.
