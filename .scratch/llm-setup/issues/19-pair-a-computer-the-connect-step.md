# 19 — Pair a computer: the Connect step

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 18
**Decided in:** ticket 13, ticket 15, ticket 16

## What to build

Build the first step of the pair-a-computer dialog in the locked design: step tabs, per-OS install commands for `cloudflared` as a service with the computer's tunnel token, a live *this computer ↔ Nexul* panel, and two checks (tunnel online, T3 Code answering) pushed live. An alert card explains a missing prerequisite with its fix (Cloudflare not connected, Zero Trust not enabled). Next stays disabled until both checks pass. Source every piece from the locked component map for this dialog (kept outside the repo; see ticket 13): registry components for the panel, checks, and command blocks, and the dialog frame and step tabs rebuilt from their reference blocks rather than a plain dialog.

## Acceptance criteria

- [ ] A new computer shows its commands, waits, and flips to connected once the tunnel and the harness answer
- [ ] Missing prerequisites show their alert card instead of a dead end
- [ ] The dialog is a full-screen sheet at 320–414px

## Surfaces

- HTTP gateway route for create and status
- MCP tool to start pairing and read status
- Live WebSocket push of the two checks

## Read first

`practices/react-guide.md` (F1–F7), `practices/design-language.md`, `practices/typescript.md`, `practices/testing.md`, `practices/go.md`, `practices/architecture.md`, and the decision tickets above.

## Verification

`bun run lint && bun run typecheck && bun run test` in `web/`; checked at 320, 375, 414, and 768px; `go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `web/src/components/settings/ (new pairing dialog)`
- `web/src/hooks/PairingHooks.tsx`
- `internal/pairing/handler.go, mcp.go`

**Size:** M
