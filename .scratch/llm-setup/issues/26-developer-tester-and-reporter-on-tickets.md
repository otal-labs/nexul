# 26 — Developer, Tester, and Reporter on tickets

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 09, ticket 10

## What to build

Rename assignee to developer, add an optional tester, and add a reporter set once at creation: a person, or Nexul on behalf of a person when a play, @Agent, MCP call, or automation filed it. Cards show the developer, or the tester in testing-stage columns. Add a "waiting for me to test" filter.

## Acceptance criteria

- [ ] Every create path sets the reporter correctly
- [ ] Cards switch avatar by stage
- [ ] The tester filter lists only tickets in testing stages

## Surfaces

- HTTP, MCP ticket tools, events, live push, search facets if indexed
- Docs: CONTEXT term already added

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/tickets/model.go, usecase.go, mcp.go, events.go`
- `internal/platform/storage/`
- `web/src/components/board/, web/src/hooks/TicketHooks.tsx`

**Size:** M
