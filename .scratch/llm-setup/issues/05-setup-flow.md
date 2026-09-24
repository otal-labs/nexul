# 05 — The setup flow itself

**Type:** grilling
**Status:** resolved
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

## Answer

Grilled with the owner 2026-09-22.

- **Nexul drives the setup turn.** The wizard is a dialog (the same family
  as the Add runner dialog) opened from the computer's row in pairing
  settings; it fires a built-in setup turn through
  the paired harness, whose instructions walk the install and verification.
  **The setup turn is the one run exempt from the unconfirmed-provider
  block** — its only purpose is to end that state. The user never configures
  MCP by hand.
- **Coarse pre-selections for now**: the whole mattpocock set, both install
  locations (`~/.claude/skills/` and `~/.agents/skills/`) ticked by default,
  covering five of six drivers. The set is one coherent loop, so no
  per-skill unticking; per-skill choice is what amending installed copies
  already covers.
- **Evidence that earns the confirmation**: files present in each install
  location *and* the harness re-reporting its discovered skills per driver
  (T3 Code exposes each driver's skills list — the ground truth of what the
  CLI will load). No trial invocation per provider. The confirm tools record
  what was verified (the skills seen), so "confirmed" is auditable.
- **Re-runs**: the wizard is re-enterable any time and idempotent —
  re-running on a confirmed computer re-verifies and re-confirms. It is the
  same surface after an un-confirm or when a new provider appears at false.
- **Instructions live in the wizard**; nexul.io gets one short page that
  points at it rather than duplicating steps.

## Comments

Amended by the MCP-surface grilling (2026-09-24): the wizard's first step now
connects Nexul's MCP server to each provider with a dedicated per-computer
token, and the wizard runs one setup turn per provider so each confirms
itself. The token is hidden in the saved transcript. See
[The setup MCP surface](12-setup-mcp-surface.md). How a remote Nexul reaches
the harness at all is open in tickets 14 and 15.
