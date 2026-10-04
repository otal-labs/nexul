# 12 — Walkthrough on a real install

**What to build:** Nothing new: walk the whole loop on an install with a
paired computer. A Restricted member who can edit only one doc writes it;
a developer runs Clarify via AI; the member answers in the browser and on
the phone, including "Anything else?"; rounds repeat until no gaps are left;
the doc is rewritten in the member's words; the developer closes it and
runs To tickets via AI. Check the member never sees agent wording, a trail,
or the thread, and that each notification arrives once. File what breaks as
fix-up tickets here.

**Blocked by:** 08, 09, 10, 11

**Status:** ready-for-agent

## Comments

- 2026-10-04, web half walked on this machine: master in an Incus box, a
  throwaway T3 Code server with Claude, a Restricted member as the client,
  four rounds to no gaps, then reopen, Stop, and the edge checks. Every step
  passed. Defects fixed in "Number a doc's question rounds by what they
  asked and hide agent wording from clients" (#442) and the lock-notification,
  inbox-bump, trail id, question-validation and instructions fixes (#443).
  The phone half waits on the phone ticket.
