# 03: How a role stores and edits its steps

Type: grilling
Status: open
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
