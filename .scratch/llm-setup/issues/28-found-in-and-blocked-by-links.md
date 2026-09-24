# 28 — Found-in and blocked-by links

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 09

## What to build

Add explicit ticket-to-ticket links. Found in: a bug's origin, or origin unknown. Blocked by: cleared when every blocker reaches a done-stage column, cycles refused on creation. The card shows a blocked icon with what it waits on; the ticket page shows both directions. Never stops a card moving.

## Acceptance criteria

- [ ] Links are records, created and removed through HTTP and MCP
- [ ] A cycle is refused with a clear error
- [ ] A blocked card clears the moment its last blocker is done

## Surfaces

- Events, live push, MCP tools, reverse state (remove link)

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/tickets/`
- `internal/platform/storage/queries/`
- `web/src/components/board/, tickets/`

**Size:** M
