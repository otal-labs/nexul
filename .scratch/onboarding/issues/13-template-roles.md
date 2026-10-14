# 13: Template roles

Type: task
Status: resolved
Blocked by: 08

Read first: `AGENTS.md`, `CONTEXT.md` (Welcome wizard, Template role) (Instance template), `practices/go.md`, `practices/mcp.md`, `practices/react-guide.md`, ticket 04's
answer, the spec (Template roles).

## What to build

- **The template kind:** the `role` kind with the five shipped keys and their code defaults (ticket 04's grid
  and ticks), plus added template roles with generated keys.
- **Workspace creation:** copies every template role beside Owner, the first workspace of a fresh install
  included.
- **The Roles card:** Settings → Templates → Roles, with add, edit in the role editor, Reset (shipped) and Delete
  (added).
- **MCP:** `template_get` and `template_update` cover the kind.
- **The locked-actions switch:** the role-aware UI effort's `show_locked_actions` is off for Client and on for the
  other four (`.scratch/role-aware-ui/`, its ticket 08).

## Acceptance criteria

- [ ] A new workspace has the template roles with their ticks, and editing a template afterwards leaves it alone.
- [ ] A shipped template can't be deleted; an added one can, and the workspace copies stay.
- [ ] `templates:write` gates every write. Verified at 768, 1024, 1440.

## Answer

Built in #639 and #640: the `role` template kind with the five shipped keys, every new workspace seeded with them, Settings → Templates → Roles, and Administrator without making or removing a workspace.
