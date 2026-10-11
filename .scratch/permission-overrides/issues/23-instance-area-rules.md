# 23 — Instance-area rules

**Status:** ready-for-agent

**Blocked by:** 17

**Branch:** master, a normal PR, after the switch. A new ability: no area carries a rule on upgrade.

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3, 8), `practices/mcp.md`
(sections 4, 6, 7), `practices/testing.md` (sections 3, 8, 10), ADRs 0087, 0088, 0135, 0148, the spec (Instance
areas).

An instance area belongs to no workspace and is held by a role bit in any workspace the person is in (ADR 0088). This
slice lets the instance Owner hand out one area to a role or a person without a custom role. The panel is slice 24.

## What to build

- Instance-area rules are `permission_overwrites` rows with `resource_type = 'instance_area'` and `resource_id` the
  entity key (`instance`, `sign_in_providers`, `connector:<name>`, `dns`, `templates`, `accounts`, `runners`,
  `topology`, `automations`, `integrations`, `audit`, `computer_activity`), target `role` or `person`. A role target is
  one workspace's role. The entity table lives in `internal/platform/permissions` beside `domainTable`, each entity
  naming the actions a rule may set; each connector registers its own.
- The evaluator: an Owner of any workspace is allowed and never deniable; otherwise the base (a role bit in any
  workspace), then the rules for the roles the person holds in any workspace, merged with Allow winning, then a rule
  naming the person, which beats them. Every instance-area check (`RequireAnywhere`, `HoldsAnywhere`,
  `PermissionsAnywhere` and their callers) asks for the entity it acts on.
- Only the instance Owner (the Owner role of the default workspace) writes these rules, as one function. Others get
  forbidden.
- `/api/auth/me`'s `instance_permissions` lists an action held on at least one entity. Reads that need only membership
  stay open.
- `permission_overwrite_list` and `_update` accept `resource_type: instance_area`; explain covers areas.
- `access.grant.changed` for an area reaches the instance Owner and the person or the role's holders.
  `make event-schemas`, `make live-topics`.
- `CONTEXT.md` and the guide page for instance settings.

## Acceptance criteria

- [ ] `TestInstanceAreas_BobManagesDNSButNotConnectors`, over HTTP and MCP.
- [ ] `TestInstanceAreas_OnlyTheInstanceOwnerWrites` and `TestInstanceAreas_AnOwnerIsNeverDenied`.
- [ ] `TestInstanceAreas_EveryInstanceActionHasAnEntity`.
- [ ] Table-driven `TestDecide_InstanceAreaOrder`: base, role rules from several workspaces, a person rule, every
      combination of states, with sources.
- [ ] `/api/auth/me` costs the same statements with rules on every area as with none, and the tool count is
      unchanged.
