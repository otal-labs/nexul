# 27 — Body templates on ticket types

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 09, ticket 10

## What to build

Add a body template to each ticket type, edited in project settings beside its name and colour, seeded for bug (steps to reproduce, expected result, actual result, provide screenshot), feature (why, acceptance criteria, out of scope), and task (what needs doing, acceptance criteria). The create dialog pre-fills it and swaps it on type change only while the body is untouched; the MCP create tool returns the template to fill.

## Acceptance criteria

- [ ] New projects get the three seeded templates
- [ ] Editing a template never rewrites existing tickets
- [ ] An agent creating a ticket through MCP receives the template

## Surfaces

- HTTP, MCP, events for type updates

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/workspace/ (ticket types)`
- `internal/platform/storage/workspace_repo.go`
- `web/src/components/settings/ (project ticket types)`

**Size:** M
