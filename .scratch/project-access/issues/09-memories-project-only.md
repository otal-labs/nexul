# 09 — Memories are project-only

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately
**Decided in:** tickets 02, 03, 06; spec section "Memories"; ADR 0099

## What to build

Remove workspace-scoped memories everywhere, backend, web, MCP, skill, and
docs, and check every memory through its project.

1. **Migration 0050**: delete `attachments` with a `memory_id` of a
   workspace-scoped memory, then their `memory_versions`, then the
   `memories` with `project_id IS NULL`, explicitly rather than by cascade.
   Drop `idx_memories_workspace_scoped`. No table rebuild; the use-case
   enforces the project. Upgrade test from 0049 (or the previous migration
   on master) with workspace and project memories, versions, and attachments.
2. **Use-cases** (`internal/memories/usecase.go`, `repo.go`, `handler.go`):
   `Create` and `Clone` refuse an empty project as invalid; drop
   `ListWorkspaceScoped` and the workspace half of `ListForProject` and
   `ListMemoryItems`; every read, write, delete, clone, version, and revert
   checks `RequireProject` on the memory's project instead of the workspace.
   `server/cmd/wire_gates.go` (`memoriesPermissionGate`) and the attachments
   `MemoryAccessChecker` follow. The `memories:clone` label in
   `internal/platform/permissions/permissions.go` becomes "Clone memories to
   another project".
3. **Live frames** (`server/cmd/live_audience.go`): `memoryFrame` checks
   `memories:read` on the memory's project; the `memory.deleted` payload
   gains `project_id` (additive) and `memoryDeletedFrame` checks it.
4. **Agent** (`internal/agent/prompt.go`, `pipeline.go`): the index is the
   project's memories only; the "Workspace memories:" block goes; a turn
   with no project carries none.
5. **Chat**: interview threads stay readable through `memories:read`, now
   on their project (shared with ticket 10's per-thread check; whichever
   lands second reconciles).
6. **MCP** (`internal/memories/mcp.go`): `memory_list` and `memory_create`
   drop `workspace_id` and workspace scope from inputs and descriptions;
   `project_id` is required. Tests in `mcp_test.go` follow.
7. **Skill**: `internal/platform/skills/nexul-memory.md` drops workspace
   scope; its content-derived version changes with it.
8. **Web**: `CloneMemoryDialog` loses the Workspace destination; the
   workspace memory route in `web/src/Router.tsx`, `MemoriesPage.tsx`, and
   `models/Project.tsx` goes (no redirect: the old path is a 404);
   `models/Memory.tsx`, `hooks/MemoryHooks.tsx` drop workspace scope; tests
   follow.
9. **Docs**: `website/src/content/docs/docs/guide/memories.md`, the
   Memories rows in `README.md` and `website/src/pages/roadmap.astro` ("per
   workspace or per project" becomes per project).
10. `rg -i 'workspace memor|workspace.scope|workspace-scoped' --hidden`
    leaves only hits unrelated to memories.

## Acceptance criteria

- [ ] Migration test: workspace memories, their versions and attachments
      are gone; project memories untouched
- [ ] Creating or cloning a memory with no project is invalid over HTTP and
      MCP; a forbidden actor and a missing project come first in the tests
- [ ] A restricted member (once ticket 08 lands) reads a project's memories
      only through Project access; until then the answers are unchanged for
      everyone
- [ ] A plain chat turn's prompt carries no memory index
- [ ] The MCP surface test passes; the skill version changes
- [ ] Hit every surface: UI yes (768, 1024, 1440px); HTTP yes; MCP yes;
      events: `memory.deleted` payload additive, catalog schema updated;
      live push yes; permissions yes; docs yes
- [ ] `make lint`, `make coverage`, `make sqlc-check`; `bun run lint`,
      `typecheck`, `test` in `web/`

## Read first

`practices/go.md`, `practices/architecture.md` (sections 1 to 6),
`practices/mcp.md` (sections 9 to 11), `practices/testing.md`,
`practices/react-guide.md` (F1 to F7), `practices/typescript.md`,
`practices/borrowed-practices.md`; ADRs 0056, 0059, 0099; this effort's
`spec.md` and ticket 02.

## Files likely touched

- `internal/platform/storage/migrations/0050_*.sql`, `queries/memories.sql`,
  `migration_0050_test.go`
- `internal/memories/`, `internal/attachments/usecase.go`,
  `internal/agent/prompt.go`, `pipeline.go`
- `internal/platform/skills/nexul-memory.md`,
  `internal/platform/permissions/permissions.go`
- `server/cmd/live_audience.go`, `wire_gates.go`
- `web/src/components/memory/`, `web/src/models/Memory.tsx`,
  `web/src/hooks/MemoryHooks.tsx`, `web/src/pages/MemoriesPage.tsx`,
  `web/src/Router.tsx`, `web/src/models/Project.tsx`
- `website/src/content/docs/docs/guide/memories.md`, `README.md`,
  `website/src/pages/roadmap.astro`
