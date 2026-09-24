# 30 — Per-branch overrides on branch deploy rules

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** ticket 10

## What to build

Let a branch deploy rule override the base service's settings for its branch (key = value), fulfilling the per-branch overrides ADR 0036 anticipated. Editable on the stack page's branch deploys section.

## Acceptance criteria

- [ ] A branch deployment runs with its overrides and the base keeps its own values
- [ ] Removing an override restores the base value on the next deploy

## Surfaces

- HTTP, MCP stack tools, events
- Docs: stacks-and-deploys guide

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`; `bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px.

## Files likely touched

- `internal/deploy/branch.go, model.go, usecase.go`
- `web/src/components/stack/`
- `website/src/content/docs/docs/guide/stacks-and-deploys.md`

**Size:** M
