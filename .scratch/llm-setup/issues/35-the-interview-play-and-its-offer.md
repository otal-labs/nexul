# 35 — The Interview play and its offer

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 34, 32
**Decided in:** ticket 08

## What to build

Add the project's interview as a third play target and seed the Interview play: one question at a time with a recommended answer, able to scan the codebase and then grill to verify, amending on re-runs. The project wizard's last step offers it; skipping asks "are you sure?"; a banner stays on the project until the interview exists.

## Acceptance criteria

- [ ] The play runs on the Interview page and writes the interview memory
- [ ] Skipping in the wizard confirms and leaves a banner
- [ ] A re-run amends instead of starting over

## Surfaces

- Plays catalog, MCP play run

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/plays/model.go, usecase.go`
- `web/src/components/wizard/WizardDoneStep.tsx`
- `web/src/pages/ (Interview page)`

**Size:** M
