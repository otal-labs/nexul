# 21 — Setup state and its MCP tools

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 04, ticket 12

## What to build

Store setup confirmation per ticket 04: a nullable confirmed-at on the computer for the overall state and one row per (computer, driver kind) recording the skills the harness reported. Add the MCP tools `computer_setup_get`, `computer_setup_confirm_provider`, `computer_setup_unconfirm_provider`, `computer_setup_confirm`, `computer_setup_unconfirm`, and `account_whoami`. Writes happen only through MCP; the HTTP gateway exposes reads only.

## Acceptance criteria

- [ ] Only the computer's owner can read or change its setup state
- [ ] Confirm and un-confirm publish events and push live
- [ ] No HTTP route can change a confirmation

## Surfaces

- Events (catalog plus outbox)
- Live push
- Reverse state: un-confirm

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/pairing/model.go, usecase.go, mcp.go, events.go`
- `internal/platform/storage/queries/, migrations`

**Size:** M
