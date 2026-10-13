# 22: Archive and restore a workspace

Type: task
Status: open
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- A migration adds `workspaces.archived_at`.
- Archive and restore use-cases. An archived workspace leaves every list, search and live frame.
- `workspace.archived` and `workspace.restored` events.
- Archive in the workspace Danger zone. Restore in instance settings for the instance owner.
- `workspace_update` takes `archived`.

## Acceptance criteria

- [ ] An archived workspace is not found for its members.
- [ ] Restore brings everything back.
- [ ] The integration test runs on real SQLite.
