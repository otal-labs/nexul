# 11 — The HTTP and MCP contract

**Status:** done

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

- [x] A `/me` response from before the switch, fed to the current client-core in a test, gives the same answers as
      the new shape.
- [x] Each removed MCP field fails with a schema error; writing a non-empty removed HTTP field is refused with the
      replacement named.
- [x] The MCP surface test keeps 110 tools.
- [x] `liveRequests.test.tsx`'s member-updated and `role.updated` counts are updated: a member-updated frame now costs
      one request, the project's people, since the access route it also refetched reads empty; `role.updated` holds at
      `/api/auth/me`, since a role changed in another workspace moves nothing here.

## Added by the owner (2026-10-11)

- [x] Q31: reordering projects takes `projects:create` and orders the projects the caller is in; a hidden one keeps
      its place.
- [x] Q32: `memories:template` (Edit the workspace Interview template), a workspace verb on memories; saving or
      resetting the template takes it, and its holders open it on Settings, while the read through a project stays as it
      was; migration 0102 (provisional) grants it to roles holding
      `memories:write`.
- [x] A pending login invite goes with its role (migration 0101, provisional), since the invite gives that one role
      and nothing else.

## Notes for later slices

- `rg -w` finds none of the removed fields in the built-in play instructions or the `nexul-memory` skill, so neither
  changes and no migration rewrites stored instructions; `api-and-tokens.md`, `people-and-access.md`,
  `mcp-server.md` and `interview.md` are updated.

- `permission_overwrite_*` take `project` and role targets; `channel` is left for slice 13, whose grid save writes the
  channel rules and needs a channel's workspace, the event's `channel` resource type and its audience.
- The invitation acceptance page and `POST /api/invitations/preview` still name a grant's first role only; slice 12
  shows every role (`InvitationGrant.roles` carries them on the list).
- `POST /api/workspaces/{id}/members` for a login with no account yet takes one role, as `workspace_invites` holds one.
- The web still sends `role_id`, `every_project` and `project_access`, and reads `/me` through the pre-switch
  client-core; slices 12 and 16 move them. `practices/native.md` still describes the Restricted member (slice 16).
- Settings reads the workspace Interview template with no project named, which a holder of `memories:template` passes;
  a member with only a project's `memories:read` reads it through that project, as before.
- Token scopes: a `PUT /api/memories/interview-template` and `POST /api/projects/reorder` still need the
  `memories:write` and `projects:write` scopes at the gateway, as project create needs `projects:write`; the installer
  must also hold the new bit.
