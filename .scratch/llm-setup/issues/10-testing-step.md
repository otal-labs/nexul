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
