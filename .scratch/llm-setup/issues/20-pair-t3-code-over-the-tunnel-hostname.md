# 20 — Pair T3 Code over the tunnel hostname

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 19
**Decided in:** ticket 15

## What to build

Second step of the dialog: pair T3 Code over the verified hostname with the fields pre-filled, so the user only pastes the `t3 pair` token. Keep today's URL pairing as an Advanced option for machines the server can already reach.

## Acceptance criteria

- [ ] A computer pairs end to end through its tunnel hostname
- [ ] URL pairing still works from Advanced
- [ ] Errors show inline on the field that caused them

## Surfaces

- MCP pairing tool accepts the tunnel path
- Docs: the paired-computers guide describes the tunnel as the default

## Read first

`practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, `practices/go.md`, `practices/architecture.md`, and the decision tickets above.

## Verification

`bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px; `go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `web/src/components/settings/`
- `internal/pairing/usecase.go`
- `website/src/content/docs/docs/guide/paired-computers.md`

**Size:** S
