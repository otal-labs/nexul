# 06 — Project access and private channels over MCP

**Type:** grilling
**Status:** resolved
**Blocked by:** 03, 04

## Question

Which tools carry setting a person's mode and Project access, reading who
has access to a project, making a channel private, and managing its members,
inside the tool budget (ADR 0068, ADR 0081)? Extend `account_update`,
`account_list`, `project_*`, and `conversation_*` before adding any tool.

Technical: decide it against `practices/mcp.md` and let the owner react.

Also in scope, from ticket 02 (workspace memories removed):

- `memory_list` and `memory_create` lose the workspace option, and their
  descriptions stop offering it.
- The `nexul-memory` skill (`internal/platform/skills/nexul-memory.md`)
  drops workspace scope in the same change. Its content-derived version
  changes with it, so every computer's copy refreshes through `skill_get`
  and shows "skills out of date" until setup re-runs; that is how agents
  already holding the skill learn the change.
- The server instructions gain one line: a person may see only some of a
  workspace's projects, so an agent reads not found as access, not a fault.

## Answer

Decided 2026-10-01; technical, for the owner to react to. No tool is added,
so the tool ceiling does not move.

- **`account_list`**: each workspace membership gains `every_project`
  (`role` or `none`) and, when `none`, `projects`: one entry per project with
  any access, `{project_id, project_name, allow}`.
- **`account_update`**: a `workspaces` entry gains `every_project` and
  `project_access`, a list of `{project_id, allow}` replacing the levels on
  each named project; an empty `allow` takes that project away. Omitted
  fields keep what is there, as `allow` and `deny` already do.
- **`invitation_create`**: a `grants` entry gains the same `every_project`
  and `project_access`.
- **`project_get`**: gains `access`, the Restricted members with access and
  their actions, filled only for a caller holding `members:write` in the
  workspace (the people who manage it), empty for everyone else.
- **`conversation_list`**: each channel gains `private` and, when private,
  `member_ids`. Private channels the caller is not in are absent (the Owner
  sees them all).
- **`conversation_update`**: retitled "Update channel"; gains `private`
  (switching it, with `member_ids` naming who stays when going private),
  `add_member_ids`, and `remove_member_ids` (yourself to leave). Creating a
  channel has no tool today and stays out of this effort.
- **`role_update`**: unchanged in shape; its description says a role's
  project areas reach only members whose Every project row is From role.
- Plus the memory tools, the `nexul-memory` skill, and the server
  instructions line listed above.
