# 06 — Surfaces and consequences of setup state

**Type:** grilling
**Status:** open
**Blocked by:** 04, 05

## Question

Where does setup state show, and what does an unconfirmed setup change?

- Pairing settings UI: show the overall boolean and the per-provider ones on
  each computer row? Read-only there by design (writes are MCP-only).
- ~~Block or warn?~~ **Decided in the ticket 03 grilling (2026-09-22): hard
  block.** A provider on a computer cannot be used for agent runs until its
  boolean is true. Left to design here: where the block bites (target
  resolution? turn start?), what the refusal tells the user, and how the
  user gets from the refusal to the wizard.
- Does unconfirmed setup surface through the existing pairing
  `NotConfiguredReason` reply path, so @Agent tells the user what to do?
- Events: does confirming setup publish to the event catalog (it is a state
  change other surfaces may care about)?
- Permissions: which `<domain>:<action>` gates the read and the confirm
  tools — pairing's existing vocabulary, or a new action?
