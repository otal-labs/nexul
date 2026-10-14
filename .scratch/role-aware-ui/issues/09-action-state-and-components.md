# 09: The action state and the shared action components

Type: task
Status: resolved
Blocked by: 08

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/typescript.md` (section 11), `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, `practices/native.md`, `practices/react-guide.md`, the spec.

## What to build

- `make permission-catalog`, which writes `client-core/permissionActions.ts` from `domainTable`: the
  `PERMISSION_ACTIONS` `as const` list and the `PermissionAction` union. CI checks it is current. Ticket 10 adds the
  route file to the same target. Every `permission` prop and `actionState` take that type, never `string`.
- `client-core`: `actionState` and the "Your role can't <verb>" line from the catalog's labels, with tests.
- Web: `ActionButton`, `ActionMenuItem`, `ActionInlineEdit` and `ActionDragHandle` in ticket 04's look. That
  means `aria-disabled`, `aria-describedby` and the tooltip on focus.
- Phone: the matching components, with tap for the reason and a 44pt target.
- `practices/react-guide.md` and `practices/native.md` say every write goes through them.

## Acceptance criteria

- [x] Table tests for live, locked and hidden across the switch, an entity answer, and the Owner.
- [x] Component tests for keyboard focus and the announced reason.
- [x] Verified at 768, 1024 and 1440. The emulator check is left for a scheduled run.

## Answer

- `make permission-catalog` writes `client-core/permissionActions.ts` from the server's catalog: `PERMISSION_ACTIONS`,
  the `PermissionAction` union and `PERMISSION_LABELS`. `TestPermissionActionsFile_MatchesTheCatalog` fails in the Go job
  when it is stale. The web's old `PermissionAction = string` named the verb, so it is gone.
- `client-core/actionState.ts` holds `actionState`, `cannotReason` and the `ActionGate` every component takes
  (`permission`, `projectId?`, `answer?`). An item's answer overrides the role-wide one, the Owner is live, and
  locked needs a role with the switch. `verb` replaces the catalog label in the reason. Instance-domain actions
  (`INSTANCE_PERMISSION_ACTIONS`, generated from the catalog's areas) read the auth `/me`'s `instance_permissions`,
  and the instance Owner is live for all of them.
- Web: `ActionButton`, `ActionMenuItem` (a row of a popover menu, which is what the app's menus are),
  `ActionInlineEdit` and `ActionDragHandle` in `components/action/`. A `RowAction` with a `permission` renders through
  `ActionMenuItem`; Delete role on the roles list is the one real use.
- Phone: `ActionButton` and `ActionRow` in `src/components/action/`; Redeploy on the stack screen is the one real use.
- `practices/react-guide.md` and `practices/native.md` state the rule.
