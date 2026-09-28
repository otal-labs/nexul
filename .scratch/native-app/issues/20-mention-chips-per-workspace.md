# 20 — Mention chip template per workspace

**Type:** implementation
**Status:** done
**Blocked by:** 16
**Decided in:** ticket 03

## What to build

Move the mention chip template from the instance settings row to the
workspace, keep it gated on `workspaces:write`, and render its section under
Configuration → This workspace. Every place that renders a chip reads the
template of the workspace the content belongs to. A new workspace starts with
the current default template. No data to migrate.

## Acceptance criteria

- [ ] Two workspaces can hold different templates and each renders its own
- [ ] Only `workspaces:write` holders in that workspace can change it
- [ ] The instance settings row no longer carries the template

## Surfaces

- HTTP route moves to the workspace; any MCP tool that reads or writes the template follows it
- Docs: `CONTEXT.md` if it describes the template's scope

## Read first

`practices/go.md`, `practices/mcp.md`, `practices/react-guide.md`, `practices/testing.md`, ticket 03.

## Verification

`go test ./...`, `make lint`, `make coverage`, `make sqlc-check`; in `web/`: `bun run lint`, `bun run typecheck`, `bun run test`.

## Files likely touched

- `internal/auth/handler.go`, `internal/workspace/` or `internal/tenancy/`, `internal/mentions/`
- `internal/platform/storage/migrations/`, queries
- `web/src/components/settings/MentionChipLayoutSection.tsx`, `web/src/hooks/AuthHooks.tsx`

**Size:** S
