# 06 — Surfaces and consequences of setup state

**Type:** grilling
**Status:** open
**Blocked by:** 04, 05

## Question

Where does setup state show, and what does an unconfirmed setup change?

- Pairing settings UI: show the overall boolean and the per-provider ones on
  each computer row? Read-only there by design (writes are MCP-only).
- Do plays and @Agent **block** on an unconfirmed computer, or run with a
  warning? A hard block makes the boolean load-bearing; a warning keeps it
  informational. The owner's framing ("with me it's fine but for others it
  isn't") suggests at least a visible nudge before the first play.
- Does unconfirmed setup surface through the existing pairing
  `NotConfiguredReason` reply path, so @Agent tells the user what to do?
- Events: does confirming setup publish to the event catalog (it is a state
  change other surfaces may care about)?
- Permissions: which `<domain>:<action>` gates the read and the confirm
  tools — pairing's existing vocabulary, or a new action?
