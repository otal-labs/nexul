# 15 — Setup steps for Pi and Grok

**What to build:** `internal/pairing/setup_prompt.go` gets explicit branches instead of the generic
fallback.
- Pi: before printing its steps, read the `pi` provider's `version` from `ListProviders` (`server.getConfig`
  carries it). Below 0.99 → stop with "Update Pi to 0.99 or later; earlier Pi has no MCP client". Null
  version → print the steps and let the `pi mcp` call fail. Steps: `pi mcp add nexul --url <url> --header
  "Authorization=Bearer <token>" --exposure direct`, then `pi mcp list`.
- Grok: `grok mcp add --transport http nexul <url> --header "Authorization: Bearer <token>"`.
- Fix the skill-locations comment (Pi reads `~/.agents/skills`). Antigravity keeps the generic text and is
  documented as unsupported for setup (ticket 17).

**Blocked by:** None — can start immediately

**Status:** ready-for-agent

Read first: `practices/go.md`, `practices/testing.md`, the spec.

- [x] Golden-string tests per branch; Pi 0.98 → the update message; Pi null version → steps
- [x] The confirm session stays fail-closed (existing tests)

## Comments

- `harness.Provider` gained `Version` (not on the wire). Protocol 1's `Providers()` fills it from `server.getConfig`'s nullable
  `version`; ticket 01 moves that parser to `internal/t3rpc` and must carry the field over, and protocol 2's `Providers()` must set it too.
- The Pi gate sits in `runSetupTurn`, so the failed turn carries the update message and no session starts. Retry goes through the same path.
- The Grok step removes `nexul` first (`grok mcp remove`, documented), as the Claude step does, so a re-run replaces the token. `pi mcp add`
  replaces an existing entry on its own.
- Ticket 17 documents: Pi needs 0.99 or later, Antigravity keeps the generic text and is unsupported for setup.
