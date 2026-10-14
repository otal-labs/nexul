# 22: Archive and restore a workspace

Type: task
Status: resolved
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- A migration adds `workspaces.archived_at`.
- Archive and restore use-cases. An archived workspace leaves every list, search and live frame.
- `workspace.archived` and `workspace.restored` events.
- Archive in the workspace Danger zone. Restore in instance settings for the instance owner.
- `workspace_update` takes `archived`.

## Acceptance criteria

- [x] An archived workspace is not found for its members.
- [x] Restore brings everything back.
- [x] The integration test runs on real SQLite.

## Answer

- Migration 0113 adds `workspaces.archived_at`. Archiving takes `workspaces:delete` in the workspace, which Owner holds and the Administrator template role does not; restoring and listing the archived take `instance:write`, the instance Owner's. The instance's own workspace is refused, since its Owner is the instance Owner who restores.
- An archived workspace is cut off in SQL where memberships resolve: the member's workspace list, the membership and role reads every permission check sits on, the id list scoping the inbox and projects, Team, the inbox badge, and doc and ticket search. A member finds it nowhere, and restoring only clears the column, so projects, people and roles return as they were.
- `workspace.archived` and `workspace.restored` carry the members' ids, since an archived workspace resolves no one; the live frame reaches them and whoever holds the restore. Both are bridged, in the event catalog and the SDK, and the web and phone refresh their workspace lists on them.
- Web: Archive workspace in the Danger zone, with a confirm that says how many people lose the workspace and how many projects go with it, and Archived workspaces with Restore on Settings → Instance for `instance:write`. HTTP: `POST /api/workspaces/{id}/archive`, `POST /api/workspaces/{id}/restore` and `GET /api/workspaces/archived`. MCP: `workspace_update` takes `archived`, and `workspace_list` takes `archived` to list them.
- Left as they are: computer rules still count an archived workspace as shared, so archiving never ends a rule that restoring would not bring back; automations of an archived workspace keep running.
