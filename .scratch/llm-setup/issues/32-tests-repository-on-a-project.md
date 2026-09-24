# 32 — Tests repository on a project

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 10

## What to build

Let a project attach a second repository marked tests, never deployed, so the one-deployable-repository rule holds. The project wizard's repository step asks whether tests live in the same repository or a separate one.

## Acceptance criteria

- [ ] A tests repository never produces a stack or deploy
- [ ] The wizard question records the answer for the interview

## Surfaces

- HTTP, MCP project repo tools

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/workspace/ or internal/repository/`
- `web/src/components/wizard/WizardRepositoryStep.tsx`

**Size:** S
