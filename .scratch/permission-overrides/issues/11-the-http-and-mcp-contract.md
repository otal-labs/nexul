# 11 — The HTTP and MCP contract

**Status:** ready-for-agent

**Blocked by:** 10

**Branch:** `permission-switch`.

Read first: `practices/mcp.md` (sections 4, 5, 6, 7, 8, 11), `practices/go.md` (section 16),
`practices/react-guide.md` (Live events), ADRs 0044, 0082, 0085, 0148, the spec (MCP), the research (sections 2.5,
5 and 6).

The HTTP API only grows (ADR 0082); MCP parameters change under `practices/mcp.md` section 11.

## What to build

- HTTP, every old field still answering:
  - `/api/workspaces/{id}/me` keeps `restricted`, now true for every non-Owner, and sends `projects` to everyone, so a
    phone build from before the switch reads per-project permissions correctly with no update;
  - `role_name` is the roles joined, beside `roles`;
  - the member routes take `role_ids` beside `role_id`;
  - `allow`, `deny`, `every_project` and `project_access` read as empty, and writing a non-empty value is refused with
    a message naming what replaced it.
- MCP:
  - `account_update` gets `add_roles` and `remove_roles` and loses `role_id`, `allow`, `deny`, `every_project` and
    `project_access`;
  - `account_list` and `workspace_list` carry `roles`;
  - `invitation_create` takes `roles` (at least one per workspace) and loses its per-person fields; `invitation_list`
    carries `roles`;
  - `permission_overwrite_list` and `_update` accept `project` and `channel`, `target` and `role_ids`;
  - descriptions are rewritten, including `role_update`'s (`internal/roles/mcp.go:60`).
- `rg -w` finds every removed field in the built-in play instructions (a migration rewrites stored copies if one names
  them), the `nexul-memory` skill (its version moves) and `api-and-tokens.md`, and each is updated.

## Acceptance criteria

- [ ] A `/me` response from before the switch, fed to the current client-core in a test, gives the same answers as
      the new shape.
- [ ] Each removed MCP field fails with a schema error; writing a non-empty removed HTTP field is refused with the
      replacement named.
- [ ] The MCP surface test keeps 110 tools.
- [ ] `liveRequests.test.tsx`'s member-updated and `role.updated` counts are updated.
