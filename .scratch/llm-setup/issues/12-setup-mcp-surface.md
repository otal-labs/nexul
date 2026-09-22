# 12 — The setup MCP surface

**Type:** grilling
**Status:** open
**Blocked by:** None — can start immediately

## Question

Name and shape the MCP tools the setup turn calls, one per use case
(`<verb>_<object>`), given the decisions in tickets 04 and 05:

- Read: the computer's overall state and per-driver states.
- Confirm / un-confirm a driver on a computer, taking the computer's id and
  recording the verified skills list the harness reported.
- Confirm / un-confirm the overall setup.
- How the exempt setup turn is marked so the block recognises it, and that
  nothing but the setup turn can claim the exemption.
- Which permission (`<domain>:<action>`) gates each tool — likely pairing's
  existing vocabulary, since the state is the user's own computer.
- The events each write publishes (catalog row plus outbox write).

Mostly engineering detail; bring the owner only the calls that are real
trade-offs.
