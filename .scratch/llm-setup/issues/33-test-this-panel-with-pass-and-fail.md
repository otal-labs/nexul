# 33 — Test this panel with Pass and Fail

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 26, 27, 30
**Decided in:** ticket 10

## What to build

A ticket in a testing-stage column shows where to test, its acceptance criteria, and Pass / Fail. The URL is the branch's preview, else a separate shared test environment, never the default branch's deployment or one sharing its network without overrides; with none, say so and nudge. Pass moves the card to the first done column and records the tester; Fail opens the bug template, posts to the ticket thread, and moves the card back to progress.

## Acceptance criteria

- [ ] Production is never offered as a test URL
- [ ] Fail's notes land in the thread in the template's sections
- [ ] Pass records who tested

## Surfaces

- MCP tools for pass and fail
- Events and live push on the card move

## Read first

`practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, `practices/go.md`, `practices/architecture.md`, and the decision tickets above.

## Verification

`bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px; `go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `web/src/components/tickets/`
- `internal/tickets/`
- `internal/deploy/ (URL resolution)`

**Size:** M
