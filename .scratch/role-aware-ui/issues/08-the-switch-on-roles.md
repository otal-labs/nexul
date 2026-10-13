# 08: Show locked actions, a switch on the role

Type: task
Status: open
Blocked by: 07

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/go.md`, `practices/architecture.md`, `practices/mcp.md`, `practices/testing.md`, `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

- A forward-only migration adds `roles.show_locked_actions`, off for existing roles.
- Role create, update and clone carry it; so do `role_update` and `workspace_list`.
- `/me` returns `show_locked_actions`, true when any held role has it on.
- A switch in the role editor's Workspace tab, with one line saying what it does.
- Tell the welcome wizard effort's ticket 13 to seed the template roles with it on for Junior Developer, Senior
  Developer, DevOps and Administrator.
- The ADR for the hidden-or-locked rule.

## Acceptance criteria

- [ ] Several roles: one with the switch on is enough.
- [ ] Clone carries it.
- [ ] Web test for the editor switch. Verified at 768, 1024 and 1440.
