# The audit log records changes from both adapters, for 45 days

The audit log wrapped the HTTP gateway and wrote one row for every authenticated request, reads included, and kept
every row forever. Opening the board wrote 29 rows, almost all GETs, and each one queued behind the single SQLite
writer, so a burst of 16 parallel reads spent about a second per 800 requests waiting on the storage serializer. The
MCP server wrote nothing, so an agent's changes were the one thing the log could not show.

Decision: the log records changes, from both adapters, and keeps them 45 days.

- **Changes only.** A GET, HEAD or OPTIONS request writes no row, and neither do the few POST routes that only read:
  batch reads too large for a query string (label colours, dev status, mention chips), credential and route checks
  that test settings before they are saved, the repository scan, machine discovery, and the browser's log relay.
  `integrations.Audited` holds the rule and the list, and a test walks the router's real route table, so a new route
  is classified by its method unless someone lists it. A route that writes anything stays audited, however small: a
  read marker, a voice join, a connection token.
- **MCP tool calls.** Every call to a tool that is not read-only writes one row, `mcp <tool_name>`, attributed the way
  an HTTP request is. MCP accepts only sessions and personal access tokens, both of which act as their user, so the
  row names the user and no token, as it does for the same token on the gateway. The tool's own read-only hint
  decides, the same hint clients use to skip confirmation. A call that runs and fails is recorded, as a refused HTTP
  request is; a rate-limited call never ran and is not.
- **45 days.** A loop started by the composition root deletes older rows at start and every 24 hours, 500 rows per
  write through the storage serializer, so other writes interleave with a large first purge. The period is a
  constant, not a setting, because the install takes none. Migration 0079 indexes `created_at` so the purge and the
  newest-first list walk the index instead of the table.

The trade-offs: the log no longer shows who read what, so it answers "who changed this" and not "who looked at this".
Rows written before the upgrade, reads included, age out on the same 45 days rather than being deleted at once. An
instance down for days purges on its next start.

Rejected: classifying routes by hand, which a new route would silently skip; auditing inside each use-case, which
would touch every domain for what one middleware and one hook already see; batching the HTTP appends into one write,
which keeps the reads' rows nobody asked for; and a configurable period, which no one has asked for.

Decided 2026-10-09.
