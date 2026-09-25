# MCP parity gaps

**Status:** needs-triage

Capabilities the web app offers that no MCP tool reaches. `practices/mcp.md`
section 4 says every capability stays reachable through some tool, so each
of these is a defect against that rule. Most predate the consolidation in
ADR 0068; it made them visible. Each wants a field on an existing tool first,
and a new tool only inside the budget (97 of 100 today).

- **Machines.** Renaming a machine and setting its stack root
  (`PATCH /api/machines/{id}`). A `machine_update` tool, or fields elsewhere.
- **Topology.** Moving or renaming nodes (`PUT /api/topology`, the canvas
  save). `topology_update` only adds and removes nodes and edges.
- **Automations.** Versions (pending version, merge, rollback, history),
  secrets, and run logs.
- **Pairing.** Pairing defaults, project links, and the per-computer provider
  and project lists.
- **Access.** Listing roles and the member picker, which `invitation_create`
  and `permission_overwrite_update` need ids from.
- **Trails.** Listing one play's trails across every target needs a
  trails-by-play query; `trail_list` requires a target today.
- **Ticket keys.** `play_run`, `decisions_check_run`, and the chat tools'
  `ticket_id` accept only the UUID, because those domains cannot resolve a
  key; the tickets and projects tools accept `REF-102` already.
- **Rate limits.** Tool calls are limited per actor; the HTTP gateway has no
  limit at all, so the same agent loop over `/api` is unbounded.
