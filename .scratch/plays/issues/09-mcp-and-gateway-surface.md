# 09 — MCP tools, gateway routes, and events

**Type:** grilling
**Status:** resolved
**Blocked by:** 03, 08

## Question

Agents are peers of the browser (AGENTS.md, "hit every surface"). List the
full surface so the spec is complete:

1. MCP tools, one per use-case: `play_list` (for a ticket or doc, only the
   ones the caller may run and that apply), `play_run` (`plays:run`), `trail_list`,
   `play_run_interrupt`, `play_create/update/delete`, and for the new
   memory entity `memory_list`, `memory_get`, `memory_create`,
   `memory_update`, `memory_delete`, `memory_create` (`memories:clone`),
   `memory_get`, `memory_update`.
2. Gateway routes mirroring them, and the live WebSocket topic the page
   subscribes to for the running state.
3. A play started from MCP runs on whose harness: the token's user, same as
   the browser. Confirm, and what provenance suffix the status change
   carries (`:mcp`, as executions do under ADR 0049).
4. Catalog rows for `play.run_started`, `play.run_finished`, and the
   `memory.*` topics, and the inbox rule that
   turns `memory.updated` into a notification for members with
   `memories:read`.
5. Search: are plays or runs indexed? Recommendation: no.

## Answer

Resolved 2026-09-16 with the owner (one round).

- **MCP tools**, one per use-case: `play_list` (for a ticket or doc; only
  plays the caller may run and that apply), `play_run`, `trail_list`,
  `trail_update`, `trail_list`, `play_create`, `play_update`,
  `play_delete`; `memory_list`, `memory_get`, `memory_create`,
  `memory_update`, `memory_delete`, `memory_create`, `memory_get`,
  `memory_update`. Gateway routes mirror them one to one.
- **Live topic** `play.run` carries run state changes so the ticket page,
  the doc page, and the board card update without a refresh.
- **A play started from MCP** runs on the token owner's harness, exactly as
  from the browser, and its status change carries the `:mcp` provenance
  suffix executions already use (ADR 0049).
- **Catalog rows**: `play.run_started`, `play.run_finished`,
  `memory.created`, `memory.updated`, `memory.deleted`. The inbox turns
  `memory.updated` into a notification for members with `memories:read`.
- **Not indexed for search**: plays and runs.
