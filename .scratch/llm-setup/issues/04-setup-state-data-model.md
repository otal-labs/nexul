# 04 — Setup-state data model

**Type:** grilling
**Status:** open
**Blocked by:** 01

## Question

Where does setup state live and what shape does it take?

- The brief asks for two levels: one "setup has been done" boolean, plus one
  "skills installed" boolean **per provider** (claude vs codex vs opencode
  inside one paired T3 Code). Does the overall boolean derive from the
  per-provider ones, or is it an independent assertion?
- Does the state hang off `pairing.Computer` (it is a fact about a machine,
  not a workspace)? What happens to it when the computer re-pairs, the token
  expires, or the provider list changes between handshakes?
- Ticket 01's answer matters here: if one install location covers claude and
  opencode alike, is per-provider still the right grain, or does the model
  track install locations instead and map providers onto them?
- Reverse state: MCP-only writes cut both ways — an agent must also be able
  to set a boolean back to false (machine wiped, skills removed). Confirm
  the un-confirm path exists.

Both booleans default to false and are writable only via MCP — that part is
fixed (see the map's Notes), not up for grilling.

## Comments

From the ticket 03 grilling (2026-09-22), the owner settled parts of this:
state is **per computer** (a new computer starts at false and needs its own
setup); the grain is **per provider** in T3 Code, so a newly appearing
provider simply shows up with its own false boolean; nothing ever auto-flips
to false — no skills release, no re-pair, no nightly update. Still open here:
whether the overall boolean derives from the per-provider ones or is asserted
independently by the wizard, and the concrete schema off `pairing.Computer`.
Note the tension with ticket 01's finding (install locations are shared:
`~/.claude/skills/` serves two providers) — per-provider booleans mean one
install can justify confirming two providers at once, and that is fine.
