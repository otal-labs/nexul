# 24 — Setup turns per provider

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 20, 22, 23
**Decided in:** ticket 03, ticket 05, ticket 12, ticket 13

## What to build

The server side of the wizard's Set up step: a use-case that starts one setup turn per provider on the computer, the only path that resolves a target without the setup gate. Each turn's instructions write Nexul's MCP entry into its provider's config with the computer's token, install the default skill set into both install locations plus the nexul-memory skill, check the files and the harness's discovered-skills list, and confirm the provider through MCP; the last one confirms the computer. Turn progress streams as events for the dialog. Re-running is idempotent.

## Acceptance criteria

- [ ] A fresh computer ends with every provider confirmed and a play runs
- [ ] One provider failing leaves the others confirmed and is retryable alone
- [ ] Re-running on a confirmed computer re-verifies without breaking anything

## Surfaces

- MCP tool to start setup
- Events per provider turn for live push

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/pairing/ (setup-turn use-case)`
- `internal/plays/ or internal/agent/ (turn instructions)`

**Size:** M
