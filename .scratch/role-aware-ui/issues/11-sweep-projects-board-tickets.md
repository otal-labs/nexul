# 11: Sweep: projects, board and tickets

Type: task
Status: resolved
Blocked by: 10

Read first: `AGENTS.md`, `CONTEXT.md` (Locked action, Source), `practices/react-guide.md` (F1 to F7 and the self-review), `practices/design-language.md`, `practices/testing.md`, the spec.

## What to build

Move every write action in research 01's projects and board, and tickets, tables onto the action components. That
includes the project wizard's steps against their real permissions.

## Acceptance criteria

- [x] A read-only role's test sees no live write action in these areas.
- [x] A role with the switch on sees them locked. Verified at 768, 1024 and 1440.

## Answer

Swept 2026-10-14:
- **Project settings**: categories, status columns, ticket types and their colors and menus, label colors, repositories, services, the Setup card and the services door.
- **Board**: Create menu, column grip and add button, ticket card drags (a reader's card only opens).
- **Ticket page**: the rail's property rows, labels, development, links, attachments, Pass and Fail, bugs, and the Delete menu.
- **Doors**: New project (switcher, empty state), New service and Deploy this repository.
- **Wizard steps**: each step against its own permission. Info needs `projects:create`, Who's in it needs `projects:access`, Repository's scan needs `projects:create`, Service needs `stacks:write` (and `deploys:write` when it deploys), Environment needs both, and Deploy branches needs `stacks:write`.

Skipped:
- The phone, which is ticket 14.
- The sidebar's New project button, which is navigation to the wizard.
- A ticket's thread and comments, which are chat.
- The doc, memory and plays rows that share the list pane's menu.

Notes:
- `permission` takes a list, for an action that writes through each, so the doors and the deploying steps state their needs in one gate. An item's `answer` belongs to the first permission; the rest read role-wide.
- Repository's scan takes `projects:create`, so the add-a-service door lists it too.
