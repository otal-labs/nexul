# 10 — The testing step

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

After docs, code, and review, the work should be deployed somewhere testers
can reach it — the owner wants Nexul to help everyone, not just developers.
Branch deploy rules with preview deployments already exist; what does the
testing step add on top?

- Who tests: manual testers, QA, the agent itself, or all three? What does a
  non-developer need to see (a URL, the ticket, a checklist?) to test a
  ticket?
- e2e strategy questions belong to the step-0 interview (same repo or a
  separate e2e repo, coverage) — this ticket decides what Nexul *does* with
  those answers during the lifecycle: does a ticket get a "testing" signal,
  a deploy link on the ticket, an assigned tester?
- How a failed test feeds ticket 09's bug flow, and how a passed test moves
  the ticket toward done.

## Comments

Facts and decisions to start from (2026-09-23): the board already has a
fixed `testing` stage between review and done. The bugs grilling settled
that a bug found before done moves the card back to progress, a bug found
after done is a new ticket with a required "found in" link, and agents may
file bugs as QA — reported as "Nexul · for <person>".
