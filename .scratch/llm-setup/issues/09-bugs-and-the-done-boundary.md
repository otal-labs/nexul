# 09 — Bugs and the done boundary

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

Two halves of one boundary, from the owner's steps 3 and 6:

- Bug tickets are written by humans but must always link the ticket the bug
  came from, so the fixing agent inherits the original context. Nexul links
  tickets to branches and PRs today, but has no bug→origin-ticket relation.
  What is the relation (a typed link? a parent?), where is it set, and what
  does the agent see?
- What happens when a bug is found on a ticket that is already done — the
  owner's open step 6. Proposal to grill: done stays immutable (never
  reopen — the why-index stays true), the bug becomes a new ticket carrying
  the origin link, and the origin ticket's page shows the bugs it spawned.
  The alternative is reopening, which rewrites history and breaks "a merged
  PR is the implementation record".

Also to check: the board and MCP surfaces for the relation, and whether the
existing ticket-finished rule (linked PR merged, none open) needs any change.
