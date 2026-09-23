# 11 — Done as an index of the why

**Type:** grilling
**Status:** open
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
