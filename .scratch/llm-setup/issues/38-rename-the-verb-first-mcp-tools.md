# 38 — Rename the verb-first MCP tools

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 12

## What to build

Rename the remaining verb-first MCP tools to `<object>_<verb>` (`search_docs`, `search_tickets`, `list_dead_letters`, `replay_dead_letter`, the invitation tools, the account tools) and update every prompt, doc, and test that names them.

## Acceptance criteria

- [ ] No registered tool name starts with a verb
- [ ] The MCP server guide lists the new names

## Surfaces

- Docs: website MCP guide

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/*/mcp.go, internal/mcp/registry.go`
- `website/src/content/docs/docs/guide/mcp-server.md`

**Size:** S
