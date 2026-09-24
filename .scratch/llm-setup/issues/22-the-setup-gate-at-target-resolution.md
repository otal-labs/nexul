# 22 — The setup gate at target resolution

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 21
**Decided in:** ticket 03, ticket 06, ticket 12

## What to build

Enforce ADR 0063 where every agent run resolves its computer and provider: a run needs both the computer's and the provider's confirmation, mapping the stored provider instance to its driver kind (an empty provider resolves to the harness default first). Add a not-configured reason with the @Agent reply naming the provider and computer and linking to that computer's wizard; plays fail on press with the same message. Provider pickers tag unconfirmed providers "needs setup" and keep them selectable. Only the wizard's own setup-turn path resolves without the gate.

## Acceptance criteria

- [ ] @Agent and plays both refuse on an unconfirmed provider with a link to the wizard
- [ ] Pickers show the tag without hiding the provider
- [ ] No request argument can bypass the gate

## Surfaces

- Refusal copy in chat and in the play run dialog
- MCP play run returns the same refusal

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/pairing/usecase.go (ResolveTarget, ResolveTargetOverride)`
- `internal/agent/pipeline.go`
- `web/src/components/play/, settings/Harness*Field.tsx`

**Size:** M
