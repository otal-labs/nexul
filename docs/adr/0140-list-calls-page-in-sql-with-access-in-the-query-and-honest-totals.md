# List calls page in SQL, with access in the query and honest totals

32 MCP list tools read their whole set, filtered it in the adapter, and sliced a page in memory. `message_list` read
every message of a conversation to return 50: on a heavy copy of a real instance, 20,053 rows and 200ms for one page.
`ticket_list` read all 5,000 tickets for each page, and every uncleared blocker in the instance. Where a scan cap kept the
read bounded, the answer was wrong past it: `notification_list` and `dead_letter_list` reported a total of 1,000 and
`has_more` false with thousands more behind them, a ticket search stopped at 200 matches and a doc search at 1,000. The
unread, archived, folder, stack, status and play filters lived in the adapters, so the HTTP gateway had none of them
(ADR 0019).

Decision: a list call reads one page in SQL, filters by access in the same statement, and reports a total that counts
exactly what the pages hold.

- **The list call.** A list use-case takes its filter and a `paging.Window` (an offset and a limit, clamped to the
  default 50 and the maximum 100) and returns the window's items and the total of the filtered list. The MCP tool shapes
  `items`, `total`, `has_more` and `next_offset` from them with `mcptool.PageOf`; its schema is unchanged. An HTTP
  endpoint that serves the same data calls the same use-case: the inbox gained `offset` and `unread_only`, trails
  `play_id`. Endpoints that return a full list keep doing so.
- **Paging in SQL.** The repo reads the window with `LIMIT` and `OFFSET` over an index ordered by the sort key and then
  `id`, and counts the same filters. A list with optional filters is one hand-written statement that reads the page's
  ids, and its rows come back by id through sqlc. Offset and not keyset, because `next_offset` is the public contract:
  `OFFSET` over an index ending in the tiebreaker skips index entries, not rows, and never repeats a row between pages
  while nothing is written. A row inserted ahead of the offset shifts later pages by one, which `message_list` says.
- **Access in the query.** `access.CallerProjects` answers which projects `RequireProject` lets the caller through for
  an action, `ProjectsAnywhere` the same for a named person with the workspaces they belong to, and the list filters
  with `project_id IN` that set. A doc's own overwrite is no project set, so `access.DocsWith` adds the docs whose
  overwrite turns the project's answer, allowed outside the set or denied inside it, decided by the rule `CanDocs` uses.
  A private channel or a DM is read by its members only and stays a per-row check.
- **Honest totals.** `total` is what the pages hold. A per-row check that has to drop rows runs over every candidate
  before paging, never on a page after `LIMIT`, and no list stops at a scan cap.

The trade-offs: a count walks the filtered set's index on every call, so a page's cost grows with the matching rows
rather than the page, though it reads no row it does not return. The inbox count looks up each notice's subject to learn
its project, unless the reader opens every project there is. A search ranks every match, as relevance requires. The page
and its count are two statements, so a commit between them can make them differ by the rows it wrote.

Rejected: a keyset cursor, which would change every list tool's schema and the offsets agents already pass back;
`COUNT(*) OVER ()` on the page, which computes every matching row to count them; filtering a page after `LIMIT` and
reading more until it fills, which leaves the total unknowable; and keeping the caps, which made the totals lie.

Lists that still page in memory: those that arrive whole from GitHub or Cloudflare (`pull_request_list`,
`repository_list`, `dns_zone_list`, `dns_record_list`, `dns_tunnel_list`); configuration-sized ones (`workspace_list`,
`project_list`, `account_list`, `invitation_list`, `permission_overwrite_list`, `automation_list`,
`play_list`, `stack_list`, `machine_list`, `computer_list`, `gateway_list`, `exposure_list`, `botwebhook_list`);
`mention_search`, whose results are bounded; `trail_list`, bounded by one target's runs; and `conversation_list`, whose
private channels, DMs, and threads are each checked per kind in one batch before paging, as the sidebar needs the whole
list anyway.

Amends ADR 0135, whose `ProjectsWith` the list calls reach through `CallerProjects`, `ProjectsAnywhere`, and
`DocsWith`. Decided 2026-10-09.
