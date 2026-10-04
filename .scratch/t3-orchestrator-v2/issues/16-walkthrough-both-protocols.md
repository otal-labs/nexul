# 16 — Walkthrough on both protocols

**What to build:** The owner walks the effort end to end on his own paired computer, on T3 Code's nightly
only (he does not run stable), with a Nexul release that carries tickets 01–24. No Incus box and no provider
token; his own T3 and providers are the test bed. Protocol 1 stays covered by its tests.

- Update T3 Code to the nightly. The computer switches to the new client with no re-pair, the settings row
  shows it connected with the nightly version, and the first turn posts no drift warning.
- Chat: a mention and a follow-up; a mention while the first runs queues and gets its own reply; an old
  thread's first turn after the update still knows its context.
- A ticket with an image, run as a play: the agent sees it.
- A play question answered live, and one answered after a Nexul restart.
- Stop on a running and on a queued turn.
- An agent that hands work to another agent: the reply waits, carries a pill, and the pill opens the
  helper's conversation; a Claude subagent gets a pill too; Stop during the wait leaves nothing running in T3.
- The phone app shows the same pill (with a build that has ticket 19): open a real hand-off reply after the
  turn ends, and scroll a long one (close to 200 steps) to its end.
- Optional: a computer setup turn on the nightly confirms.

Findings become fix-up tickets here. Going back to stable is not part of the walkthrough; its message is
covered by tests.

**Blocked by:** 13, 14, 19, 20

**Status:** ready-for-human

- [ ] Every step passes or has a fix-up ticket
