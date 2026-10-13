# 08: A role carries its welcome steps

Type: task
Status: resolved
Blocked by: 07, and the permission rework's `permission-switch` branch merged

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/react-guide.md`,
`practices/testing.md`, the spec (Decisions 6 and 7; Records; HTTP and MCP).

## What to build

- **Migration:** the spec's migration, all three records: `roles.welcome_steps`, `welcome_step_states`, and
  `enrolled_by` on enrollment codes and runners.
- **Roles:**
  - Create, update and clone carry `welcome_steps` and validate the keys. `roles:write` is the only check.
  - The Owner role answers `github`, `computer` from code.
  - `role.updated` gains `welcome_steps_added`; run `make event-schemas`.
- **HTTP and MCP:** the role request takes the list, `role_update` takes it, and `workspace_list` returns it.
- **Web:** the role editor's **Welcome** tab after In projects. It holds five switches with one line each, and
  shares the draft, the dirty check, the discard confirm and Save.
- **ADR:** "a skip lasts until the step is given again, and template roles are copied once".

## Acceptance criteria

- [ ] Saving a role with a new tick publishes the added key; an unknown key is refused.
- [ ] Cloning copies the ticks. The Owner role refuses edits and reports its fixed steps.
- [ ] Web tests for the tab's dirty and save paths; verified at 768, 1024, 1440.

## Answer

Built in #631:
- migration 0106 (`roles.welcome_steps`, `welcome_step_states`, `enrolled_by` on enrollment codes and runners)
- roles carrying the steps on every surface
- the Welcome tab
- ADR 0149
