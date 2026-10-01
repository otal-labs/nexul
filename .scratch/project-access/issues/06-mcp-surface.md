# 06 — Project access and private channels over MCP

**Type:** grilling
**Status:** open
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
