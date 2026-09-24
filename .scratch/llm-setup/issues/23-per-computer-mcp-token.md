# 23 — Per-computer MCP token

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 21
**Decided in:** ticket 12

## What to build

Mint a personal access token per computer, "Nexul MCP on <computer>", listed and revocable on the computer's row. Un-confirming the computer's setup or removing the computer revokes it. The token is hidden wherever a transcript is saved.

## Acceptance criteria

- [ ] Minting, listing, and revoking works from the row and through MCP
- [ ] Un-confirm and unpair revoke the token
- [ ] A saved transcript never contains the token

## Surfaces

- Tokens page shows the per-computer token
- Reverse state: revoke

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/auth/ or the personal-access-token use-case`
- `internal/pairing/`
- `internal/plays/run.go (transcript redaction)`

**Size:** S
