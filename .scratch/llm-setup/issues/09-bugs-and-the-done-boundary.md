# 09 — Bugs and the done boundary

**Type:** grilling
**Status:** resolved
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

## Answer

Grilled with the owner 2026-09-23. Standing principle from the owner: few
restrictions, room for mistakes — signals over gates.

- **"Found in" link**: an explicit ticket-to-ticket record from a bug to the
  ticket it came from. The **Bug type requires it** and opens with a steps /
  expected / actual template. Moved here from the parked
  `.scratch/ticket-workflow-depth/` spec.
- **Done is the boundary.** A bug found before done (review or testing) just
  moves the card back to progress — still in flight, same PR loop. A bug
  found after done becomes a new ticket with a "found in" link; done is
  never reopened, so the done ticket's why stays true, and its page lists
  "Bugs found after done".
- **The fixing agent gets one hop**: a play on a bug also receives the
  origin ticket's body, its doc, and its linked PRs — not the origin's
  origin.
- **Humans and agents both file bugs** (agents can act as QA): a **Report a
  bug** button on any ticket pre-fills the link, the create dialog requires
  it, and an MCP tool takes it.
- **Reporter on every ticket** (tickets record no creator today): a person
  when a person filed it; **Nexul**, shown like a user with the person it ran
  for beneath ("Nexul · for Onik"), when a play, @Agent, or MCP call filed
  it; Nexul with the automation's name when an automation did.
- **"Blocked by" links** (the owner's example: frontend `/books` blocked by
  backend `/books`), also moved from the parked spec: they clear by
  themselves once every blocker reaches a done-stage column; cycles are
  refused on creation; **no blocked stage** — the card shows a blocked icon
  and what it waits on, the ticket page shows both directions. Not a gate:
  cards still move freely; plays ask "are you sure?" on a blocked ticket,
  and the agent's context lists each blocker and whether it is done.
