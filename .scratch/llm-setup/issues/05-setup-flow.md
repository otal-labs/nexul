# 05 — The setup flow itself

**Type:** grilling
**Status:** open
**Blocked by:** 03, 04

## Question

How does the installing agent actually run, and what earns the confirmation?

- Chicken and egg: plays and the loop assume the skills are already there,
  so setup can't be "run a play". Does the new user point their own harness
  (their local Claude Code / T3 Code chat) at the Nexul MCP and follow a
  workflow prompt? Paste a documented bootstrap prompt? Something else?
- What evidence must the agent gather before calling the confirming tool —
  files present on disk, a trial skill invocation, a provider handshake?
  The whole point of MCP-only writes is that the LLM *assessed* the setup;
  define what assessment means so the confirmation is worth something.
- Re-run story: how does a user redo setup after a new machine, a new
  provider, or an un-confirm (ticket 04's reverse state)?
- Where do the human-readable instructions live (nexul.io docs, the pairing
  page, an MCP prompt) so a brand-new user finds the flow at all?
