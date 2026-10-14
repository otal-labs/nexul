# 02: The invite page before and after sign-in

Type: task
Status: open
Blocked by: 01

Read first: `AGENTS.md`, `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec (Decisions 2 and 3, Look: the invite page, Architecture). The prototype
is on `origin/proto/invitations`; read it, never merge it.

## What to build

- **Preview**: adds `instance_name` and role descriptions, and returns no project names.
- **Acceptance**: adds role descriptions and `projects` per grant, through a role → projects lookup that costs one
  statement for all the grant's roles.
- **The page**: the invite page in the Consent look, before and after sign-in, with Decline. It replaces
  `InvitationGrantSummary`'s chips and the "roles and overrides" line.

## Acceptance criteria

- [ ] The preview never carries project names, and a test pins it.
- [ ] The acceptance lists the projects that list any of the grant's roles, and a statement guard holds for many roles.
- [ ] Web tests cover both states and a role with no description.
- [ ] Verified at 390, 768, 1024 and 1440.
