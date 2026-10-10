# 13 — Instance-area rules: a role or a person, written by the instance Owner

**Status:** ready-for-agent

**Blocked by:** 01, 02

Read first: `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2, 3, 8),
`practices/mcp.md` (sections 4, 6, 7), `practices/testing.md` (sections 3, 8, 10), ADRs 0087, 0088, 0097,
0135, 0148, the spec (Evaluation order, Instance areas).

An instance area belongs to no workspace and is held today by a role bit in any workspace the person is in
unrestricted (ADR 0088). This ticket lets the instance Owner hand out one area to a role or a person without a
custom role: "bob manages DNS but not connectors". The panel is ticket 14.

## What to build

- Instance-area rules are `permission_overwrites` rows with `resource_type = 'instance_area'`, `resource_id`
  the entity key (`instance`, `sign_in_providers`, `connector:<name>`, `dns`, `templates`, `accounts`,
  `runners`, `topology`, `automations`, `integrations`, `audit`, `computer_activity`), target `role` or
  `person` only. A role target is one workspace's role. The entity table lives in
  `internal/platform/permissions` beside `domainTable`, each entity naming the actions a rule may set; each
  connector registers its own entity with the connector.
- A guard test fails when an instance-area action in `domainTable` belongs to no entity.
- The evaluator: Owner of any workspace allowed and never deniable; otherwise the base (a role bit in an
  unrestricted workspace, with that workspace's person rule), then role rules, then a rule naming the person;
  Allow beats Deny between role rules from different workspaces, and a person rule beats role rules; a
  Restricted member gains nothing from a role rule, only from one naming them. Every instance-area check
  (`RequireAnywhere`, `HoldsAnywhere`, `PermissionsAnywhere` and the callers of each: DNS, connectors,
  instance, templates, accounts, runners, topology, automations, integrations, audit) asks for the entity it
  acts on. A sign-in provider action asks the `sign_in_providers` entity, a URL or upgrade action the `instance`
  entity.
- Only the instance Owner, the holder of the Owner role of the default workspace, writes these rules, and
  holds what they give. Others get forbidden. This is one function so a later definition changes in one place.
- `/api/auth/me`'s `instance_permissions` lists an action when the person holds it on at least one entity.
  Reads that need only membership today stay open.
- `permission_overwrite_list` and `_update` accept `resource_type: instance_area`; explain covers areas, naming
  the rule, or the role and workspace that hold the bit, or the Owner. A forbidden answer on an area carries
  the source sentence.
- `access.grant.changed` for an area reaches the instance Owner and the person or the role's members.
  `make event-schemas`, `make live-topics`.
- `CONTEXT.md` and the guide page for instance settings.

## Acceptance criteria

- [ ] `TestInstanceAreas_BobManagesDNSButNotConnectors`: a person with no instance bit and an Allow on DNS
      writes records and is refused on a connector, over HTTP and MCP.
- [ ] `TestInstanceAreas_OnlyTheInstanceOwnerWrites`: an Owner of another workspace, an admin and a role holding
      every bit but not the Owner role are refused.
- [ ] `TestInstanceAreas_AnOwnerIsNeverDenied`: a rule naming an Owner changes nothing.
- [ ] `TestInstanceAreas_RestrictedGainsOnlyFromARuleNamingThem`, and a role rule can still deny them.
- [ ] `TestInstanceAreas_NoEveryoneTarget`: an Everyone write is refused.
- [ ] `TestInstanceAreas_EveryInstanceActionHasAnEntity`.
- [ ] Table-driven `TestDecide_InstanceAreaOrder`: base, role, person, several workspaces, every combination of
      states, with sources.
- [ ] The matrix diff on a production copy is empty, `/api/auth/me` costs the same statements with rules on
      every area as with none, and the tool count is unchanged.
