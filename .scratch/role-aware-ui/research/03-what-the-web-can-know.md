# 03: What the web can know about one thing's permissions after the switch

Paths are on `origin/master` unless marked. `permission-switch` is already squashed into master as #614 (56e06f3e). Between the two refs, `internal/access`, `internal/platform/permissions`, `internal/tenancy`, `client-core/` and `native/` do not differ; the only differences are later web and plays commits (#615 to #617). "Before" below means the merge-base, 3d11cd32.

## 1. What the web gets today and after the switch

- **`GET /api/workspaces/{id}/me`** (`internal/tenancy/handler.go:199-222`, built by `MyAccess`, `internal/tenancy/usecase.go:341-363`) returns `role_name`, `roles[]`, `permissions`, `restricted`, `projects[]` and `access_setup_finished`.
  - `permissions` is the role union only. For a non-Owner the project-area actions are deleted from it (usecase.go:349).
  - `projects[]` lists each project the person is in, with the project-area actions held there. That answer is roles plus the project's rules, and no doc, play or stack rule (`internal/access/gate.go:327-366`, which calls `decide(..., noRules)`).
  - Before the switch, `restricted` was true only for Restricted members and `projects` was sent only to them. Now `restricted` is true for every non-Owner and `projects` goes to everyone (ticket 11). Ticket 16 is done.
- **client-core** (`client-core/permissions.ts:41-67`): `MyWorkspaceInfo`, `hasPermission`, `workspaceWidePermissions` (role union plus every project's actions) and `projectPermissions(info, projectId)`. The latter returns the role union plus that project's actions, and falls back to the workspace-wide set for an unlisted project. The `restricted` branch is gone from both helpers. Ticket 16 moved this.
- **Web hooks**:
  - `useAreaAccess(projectId?)` (`web/src/hooks/AccessHooks.tsx:12-19`) returns a function over `AREA_PERMISSION`.
  - `useHasPermission` (`web/src/hooks/WorkspaceHooks.tsx:57-62`) takes the project in view.
  - `useHasInstancePermission` (`AccessHooks.tsx:42-45`) reads `instance_permissions` from `/api/auth/me` (`internal/auth/model.go:219`): an action held in at least one workspace.
  - `AreaGate` handles routes (`web/src/components/auth/AreaGate.tsx`).
- **Phone**: `useAreaAccess` (`native/src/hooks/WorkspaceHooks.tsx:50-55`) uses the same `projectPermissions`. Its `/me` query refreshes on `role.updated` and `access.grant.changed` (WorkspaceHooks.tsx:39). No phone code calls explain.
- **Source sentences** reach the client in two ways: the forbidden message after a refused call (`explain.go:234-236`: `forbidden: <action> required (<sentence>)`), and `GET /api/permissions/explain` (see 2). `/me` carries no sentence. There is no HTTP route named "ProjectPermissions"; `Service.ProjectPermissions` (`usecase.go:248`) is internal and used by the project people card (`internal/tenancy/people.go:100`).

## 2. Is there a per-entity answer, and what do 19, 23 and 24 add?

Yes, for three entity types. `GET /api/permissions/explain?resource_type=&resource_id=&user_id=|role_id=` (`internal/access/handler.go:103-130`) wraps `Service.Explain` (`explain.go:109`). It is also reachable from MCP as `permission_overwrite_list` with `user_id` (`mcp.go:21,103`).

- **Shape** (`internal/access/model.go:114-155`): `{resource_type, resource_id, user_id, effective: [{action, allowed, source, fallback}]}`. `source` carries `kind` (owner, member, role, rule, outside_project, not_in_project), `entity_type`, `entity_id`, `target_type`, `target_id`, `state`, `entity_name`, `target_name` and `sentence`. `fallback` is the same verdict with the entity's own rule at Fallback. It is the real evaluator (`decide`) with names resolved, so it carries the source sentence.
- **Coverage** (`explain.go:18-25,41-65`):
  - doc: the docs actions plus `permissions:write`
  - play: only `plays:run`
  - project: `projects:open` plus every project-area action
  - Nothing else. A stack, machine, instance area or channel is refused as invalid. The MCP schema text names channel, but `entityOf` has no channel case.
- **Access rule**: your own answer needs only being able to open the entity; another person's needs the manage permission. An entity you can't open is not found, with no source.
- **Web use today**: `useFetchExplanation` (`web/src/hooks/PermissionHooks.tsx:42-52`). It drives the permissions panel and `useCanManageAccess`, whose comment says a doc's check costs a request, so call it from an open menu (`web/src/hooks/useCanManageAccess.tsx`).
- **Unbuilt (all `ready-for-agent`)**:
  - 19: stack rules for `stacks:*` and `deploys:*`, and explain covering stacks.
  - 22: See, Run, Edit and Delete rules per play, so explain gains `plays:read`, `plays:write` and `plays:delete`.
  - 23: instance-area rules, explain covering areas, and `/api/auth/me`'s `instance_permissions` listing any action held on at least one entity.
  - 24: the panel UI for areas, instance Owner only.
  - 21 (computer sharing) is also unbuilt and its checks are never memoised.
  - Doc-folder rules (18) and channel rules (20) were not read for this note.

## 3. "Your role can't" versus "a rule on this thing denied it"

**With explain, yes.** `source.kind` is `role` ("Not allowed: role X doesn't grant it", or "holds no role") when the role union lacks the action. It is `rule` ("Denied on doc Pricing for bob") when a rule decided it, with `entity_type` and `entity_id` saying which. `not_in_project` and `outside_project` are separate kinds.

**Without a new request, only partly:**
- For workspace-area and instance-area actions, `/me.permissions` and `instance_permissions` are the whole answer on master. No rule exists for those yet (only a play's `plays:run` deny and a channel's `channels:read` deny), so absent means "your role can't".
- For project-area actions (docs, tickets, stacks, deploys, memories, `projects:*`), `/me` gives only the combined result in `projects[].actions`. The role union is stripped for non-Owners (usecase.go:349), so a missing action could be the roles or a project rule. This cannot be told apart from `/me` alone.
- An entity's own rule (doc, play) is invisible to `/me`. A holder of `docs:write` project-wide can still be denied on one doc, and only explain or the refusal message says so.
- After 19, 22 and 23, `/me` will be even less able to answer for a stack, a play or an area.

## 4. Fetch cost per list row

- The server keeps lists flat. `TestStatements_ListsCostTheSameAtAnyLength` (`server/cmd/access_memo_test.go:284-341`) pins identical statement counts at 2 and 12 rows. `TestStatements_RoleRulesCostTheSameAtAnyLength` (:467) and `...ProjectRulesCostTheSameAtAnyLength` (:507) pin the same for doc and project rules. `TestRows_ListToolsReadOnlyTheirPage` (`server/cmd/list_paging_test.go:88`) holds a page near 250 rows over 3,000. Lists filter by read access in SQL, so unreadable docs leave the list (`CanOpen` is always true, `internal/docs/model.go:68`).
- No list row carries write or delete answers for the viewer.
- One explain call per row is N HTTP requests. Each reads layers and the entity's rules, and resolves names. No statement guard covers explain, and the spec calls it "off the hot path" (`spec.md:420`). Statement cost per call is unverified.
- Per-row answers need no fetch where `/me` suffices: project-level actions, workspace and instance areas, and anything with no entity rule. They matter only for doc, play, stack and channel rows. Existing precedent is server-computed flags on a row (`can_detach` and `CanClose`), not a per-row request.
