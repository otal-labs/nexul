# 01: Every role carries a one-line description

Type: task
Status: open
Blocked by: None

Read first: `AGENTS.md`, `CONTEXT.md` (Role, Template role, Invitation), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec (Decisions 1, Look: the role editor, Architecture).

## What to build

- **Migration**: `roles.description`, empty for existing roles, with no backfill.
- **Roles carry it**:
  - on create, update and clone
  - in `role_update` and `workspace_list`
  - in `tenancy.TeamRole` and `auth.InvitationRole`
  - in `role.updated`, then `make event-schemas`

  It is capped at 140 characters.
- **Template roles**: `TemplateBody.description`, and text for the five shipped roles.
- **Role editor**: the Description field from the spec.

## Acceptance criteria

- [ ] A migration test on a production-shaped database leaves every existing role with an empty description.
- [ ] A PATCH without `description` keeps it, and one over 140 characters is refused.
- [ ] A new workspace's template roles carry their descriptions.
- [ ] Web tests cover the field and its counter. Verified at 768, 1024 and 1440.
