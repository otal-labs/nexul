# 02: Machine rules

Type: task
Status: open
Blocked by: None — can start immediately

Read first: `AGENTS.md`, `CONTEXT.md`, `practices/go.md` (sections 14, 17), `practices/architecture.md` (sections 2,
3, 8), `practices/mcp.md` (sections 4, 6, 7), `practices/testing.md` (sections 3, 8, 10), ADRs 0088, 0140, 0148,
0150, the spec (Decisions, The actions, Evaluation).

Adds the rules and the evaluator; nothing enforces them yet, so nobody's access changes in this slice.

## What to build

- Actions in `internal/platform/permissions`: `machines:see` off the grid beside `projects:open` (refused by
  `ParseAction`), `machines:deploy`, `machines:delete` and `machines:create` on it. `machines:write` stays and is
  labelled Manage.
- `resource_type = 'machine'` rules in `permission_overwrites`, `resource_id` the machine id, target a role (of any
  workspace) or a person. The machine row's delete cascades its rules (a numbered forward-only migration).
- The evaluator in `internal/access`, following the spec's Evaluation: `MachinePermissions(ctx, userID, machineID)`,
  `RequireMachine(ctx, machineID, action)`, and a batched `MachinesSeen(ctx, userID)` for list filtering. One
  statement per check, memoised per request like the other rule types.
- Writing rules: the instance Owner and Manage holders on the machine, hold-what-you-give (ADR 0088), adding a Deny
  never limited. `permission_overwrite_list`, `_update` and `GET /api/permissions/explain` accept `machine`.
- `access.grant.changed` for a machine rule reaches the instance Owner, the person or the role's holders.
  `make event-schemas`, `make live-topics`.
- `CONTEXT.md`: Machine rules, See / Deploy to / Manage / Delete.

## Acceptance criteria

- [ ] Table-driven `TestDecide_MachineOrder`: instance Owner, another workspace's Owner, role rules from several
      workspaces, a person rule, Deny beating a role, no rule; every answer with its source.
- [ ] `TestMachineRules_OnlyTheInstanceOwnerAndManageHoldersWrite`, with hold-what-you-give.
- [ ] `TestMachineRules_NoRuleMeansNotFound` for See.
- [ ] A check costs the same statements with rules on many machines as on one.
