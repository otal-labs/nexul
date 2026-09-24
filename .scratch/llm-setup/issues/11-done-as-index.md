# 11 — Done as an index of the why

**Type:** grilling
**Status:** resolved
**Blocked by:** None — can start immediately

## Question

The owner's step 7: when a ticket is done and tested, its purpose is to be
indexed so anyone — human or LLM — can later see *why* the work happened.
Today the chain doc→ticket→branch→PR already exists and search covers doc
and ticket text. What does "indexed" add beyond that?

- Is the why already carried by the existing links plus the ticket's spec,
  and this step is just making sure agents *read* that chain (a retrieval
  concern, maybe an MCP tool that walks a ticket's full context)?
- Or does done trigger something new — a closing summary written onto the
  ticket, a memory, the trail kept?
- What should an agent starting the next loop actually receive when it asks
  "why does this code exist"?

## Comments

From the interview grilling (2026-09-23), the owner wants a **decisions
log** memory beside the interview memory: the project's history of what
changed and why. Agreed shape:

- Written **only when a ticket changes how the project works** — a new
  pattern, a dropped library, a reversed decision. Routine tickets add
  nothing, so it never becomes a changelog. (Owner agreed explicitly.)
- **Indexed, not included every turn**: it sits in the memory index and is
  pulled when relevant, since the interview memory already takes the
  every-turn slot and the per-ticket why stays on the ticket's own links to
  its doc and PR. (Proposed; the owner did not object.)

This ticket decides who writes an entry (the finishing agent, at done?) and
what an entry holds.

## Answer

Grilled with the owner 2026-09-24.

- **Every ticket reaching done gets the agent's judgement** — "did this
  change how the project works?" — whoever built it (owner: people forget).
  A built-in **decisions check** play fires by itself when a card enters a
  done-stage column. It runs on the paired computer of the person who moved
  the card (like any person-started play); when a merged PR moves the card,
  on the ticket's Developer's computer. It reads the ticket, its PR, and the
  current decisions log, then writes an entry, supersedes one, or does
  nothing. If that person has no confirmed computer it never fails silently:
  the done ticket shows "Decisions check didn't run" with a button to run it.
- **An entry** is at most three lines: date, the decision in one line, why,
  and a link to the ticket (which links its doc and PR). When a new decision
  reverses an old one, the old line is marked "superseded by …" so the log
  reads as what is true now.
- **"Why does this code exist"** is one MCP tool: given a commit or PR
  number (from `git blame`; every master commit carries its PR number), it
  returns the PR, its tickets, their docs, bugs found after done, and the
  decisions-log entries citing those tickets. Its name is settled with the
  rest of the MCP surface.
- **Done writes nothing else** — no closing summary. The PR description,
  ticket, thread, and play transcript already hold what happened; a summary
  would be another copy that drifts.
- The decisions log itself stays as agreed in the interview grilling: a
  project memory, entries only for tickets that change how the project
  works, pulled from the memory index rather than sent every turn.
