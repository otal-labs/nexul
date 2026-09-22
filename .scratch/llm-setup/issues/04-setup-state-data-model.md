# 04 — Setup-state data model

**Type:** grilling
**Status:** resolved
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

## Answer

Grilled with the owner 2026-09-22; a re-verification of driver skill support
landed the last piece.

- **Overall boolean: stored, not derived.** The wizard's last step sets it
  via MCP; the UI only ever reads. A user may finish the wizard while
  skipping a provider they never use — the per-provider block covers the
  skipped one.
- **Per-provider state keyed by driver kind** (`claude`, `codex`, `opencode`,
  `cursor`, `grok`, …), never by T3 instance id (ids churn on reinstall; two
  instances of one driver share the same skills on disk).
- **No exempt category.** The re-verification (see the research addendum in
  [research/skill-discovery-per-provider.md](../research/skill-discovery-per-provider.md))
  proved the owner right: Cursor and Grok both discover skills natively —
  Cursor scans `.cursor/`, `.claude/`, `.agents/`, and `.codex/` skill dirs;
  Grok reads `~/.grok/skills/` plus `~/.agents/skills/`. Every driver is
  skill-capable, so every provider gets a boolean and the hard block is
  universal. The minimal two locations (`~/.claude/skills/` +
  `~/.agents/skills/`) cover five of six drivers; Antigravity alone has a
  separate mechanism.
- **Honor system** on the confirm call: it takes the computer's id, and the
  LLM's assessment is the trust anchor — no machine attestation. Add a
  handshake only if real misuse appears.
- **The un-confirm path exists**: the same MCP surface sets a boolean back to
  false; it is the only way down (nothing auto-flips, per ticket 03).
- Schema sketch for the implementation tickets: nullable `confirmed_at`
  timestamps rather than raw booleans (same read, richer audit) — one column
  on the computer for the overall state, one row per (computer, driver kind)
  for the providers, written only by the MCP use-cases.
