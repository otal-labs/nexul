# 34 — Interview memory, Interview page, and template

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 08

## What to build

Give each project an interview memory, included in full in every agent turn in the project and not switchable off (ADR 0065), with a length cap. Add the Interview page that views and edits it, and the workspace Interview template in workspace settings seeded with the starting categories.

## Acceptance criteria

- [ ] Every play and @Agent turn in the project carries the interview memory
- [ ] Editing the page versions the memory like any other
- [ ] A new project starts from the workspace template

## Surfaces

- Memory versioning and revert
- MCP memory tools cover it

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/memories/`
- `internal/agent/prompt.go`
- `internal/plays/run.go`
- `web/src/pages/ (Interview page)`

**Size:** M
