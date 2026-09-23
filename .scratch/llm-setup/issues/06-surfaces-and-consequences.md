# 06 — Surfaces and consequences of setup state

**Type:** grilling
**Status:** resolved
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

## Comments

From the setup-flow grilling (2026-09-22): the wizard's built-in setup turn
is the one run exempt from the block, so wherever the block bites it must
let a setup-marked turn through (ticket 12 defines the marking). The refusal
should lead straight to the wizard, which is re-enterable from the
computer's row in pairing settings.

## Answer

Grilled with the owner 2026-09-23 (the owner may revisit the placement
later; this is the working cut).

- **A run needs both booleans true**: the computer's overall setup *and* the
  provider's. Covers an agent that confirmed a provider but died before the
  wizard finished.
- **The block bites at target resolution.** @Agent and plays both resolve
  their computer and provider through one pairing function
  (`ResolveTarget` / `ResolveTargetOverride`), so one guard there covers
  every path, current and future, before anything reaches the harness. The
  stored provider is a T3 instance id while setup state is keyed by driver
  kind, so the guard maps instance to driver (the harness's provider list
  carries both); an empty provider resolves to the harness default first.
- **Refusal**: a new not-configured reason beside the existing ones, so
  @Agent replies with e.g. "@Agent can't use Codex on *Onik's laptop* until
  setup is done — run setup from Settings → Pairing", linking straight to
  that computer's wizard. Plays keep their buttons enabled; pressing one on
  an unconfirmed target fails with the same message and link. No greying
  ahead of time.
- **Pairing row**: a "Setup: confirmed / not set up" badge per computer, one
  line per provider with its state and confirmed-at time, and a **Set up** /
  **Re-run setup** button opening the wizard dialog. Nothing on the row
  changes state directly.
- **Provider pickers** (project link, defaults, play run options) list
  unconfirmed providers with a "needs setup" tag, still selectable; the run
  is what is blocked. Hiding them would make configured defaults vanish.
- **Permissions**: owner-only — a paired computer belongs to one user and
  pairing has no permission rows, so no new permission. **Events**: confirm
  and un-confirm publish (catalog row plus outbox write) and push over the
  WebSocket, so the row updates live while the wizard runs.
