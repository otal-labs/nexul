# 29 — Bug flows

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 26, 27, 28
**Decided in:** ticket 09, ticket 10

## What to build

Wire bugs per ADR 0064: the bug type requires found-in unless the origin is marked unknown; the create dialog offers feature and task only; Report a bug on any ticket pre-fills the link and the board's own Report a bug allows origin unknown; a done ticket lists bugs found after it. A play on a bug receives one hop of origin context; a play on a blocked ticket asks "are you sure?" and the agent is told each blocker and whether it is done.

## Acceptance criteria

- [ ] Every way to file a bug respects the found-in rule
- [ ] The one-hop context appears in a bug play's prompt
- [ ] Blocked plays confirm before running

## Surfaces

- MCP create accepts bugs on the same terms
- Agent prompt context

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/tickets/`
- `internal/plays/run.go`
- `web/src/components/tickets/, board/`

**Size:** M
