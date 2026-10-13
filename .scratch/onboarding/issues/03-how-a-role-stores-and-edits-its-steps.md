# 03: How a role stores and edits its steps

Type: grilling
Status: resolved
Blocked by: 01

## Question

Each role ticks which Welcome wizard steps its holders see. How is that held
and edited, across every surface?

- The field on the role: a JSON list of step keys beside `permissions`, or
  something else. It needs a new forward-only migration that sets the Owner
  role's ticks on existing workspaces.
- The role editor (`RoleEditorDialog.tsx`): a fourth tab next to Instance,
  Workspace and Every project, which are permission areas while this one is
  not. Its draft, dirty check and save.
- The HTTP role request and the MCP `role_update` tool.
- Who may change a role's steps: the same permission that edits the role?

## Answer

Owner's calls, 2026-10-13, read against the `permission-switch` branch.

- **Storage**: a JSON list of step keys on the role, next to `permissions`,
  read whole and never searched by step. It goes in a new forward-only
  migration numbered after the switch's (0099–0102 are taken there).
  Existing roles start with no ticks.
- **Keys**: `github`, `computer`, `projects`, `deploy_server`, `phone`.
  Profile is not a tick: every Welcome wizard opens on it.
- **Owner role**: it stays uneditable. Its steps are fixed in code: Connect
  GitHub and Your computer, both skippable. Anyone given the Owner role later
  goes through the Welcome wizard but never the owner wizard, so this applies
  to them too.
- **Giving a step again**: saving a role compares its old ticks with the new
  ones, and every newly ticked step clears that step's skip for the role's
  holders. Giving a person a role clears their skips for the steps it ticks.
- **Editor**: a fourth tab, **Welcome**, after In projects, in
  `RoleEditorDialog`. It holds five switches, each with one line saying what
  the holder will be asked to do. It is not a permission area. It shares the
  role's draft, the dirty check, the "discard changes?" confirm and Save.
- **Who may change ticks**: whoever may edit the role (`roles:write`). The
  "you can only grant what you hold" check does not apply, because a step gives
  no power.
- **Clone**: copying a role to another workspace copies its ticks.
- **Surfaces**: the HTTP role request and MCP `role_update` take
  `welcome_steps`, and `workspace_list` returns them with each role.
