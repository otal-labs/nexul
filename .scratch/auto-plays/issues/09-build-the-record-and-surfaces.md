# 09: Build the auto play record, permissions, HTTP and MCP

Type: task
Status: resolved
Blocked by: 02

## Question

Build what 02 decided: the migration (forward-only, production data is
live), the repo and use cases with the `autoplays` permission checks, the
HTTP routes, the MCP tool changes, events and live push, the Access page
entry, and Go tests including the error paths.

## Notes

- Backfill per 02's answer: `autoplays:read` from `plays:read`, `read`
  and `write` from `plays:write`, `delete` from `plays:delete`.

## Answer

Built in PR #524: migration 0082 (the `auto_plays` table,
`workspaces.auto_play_daily_cap`, and the backfill from the `plays` bits,
denies mapped one to one), the `autoplays` permission domain, routes under
`/api/plays/{playID}/auto-plays` and
`GET|PATCH /api/workspaces/{ws}/plays/auto-play-limits` (cap 1 to 50),
`play_list` returning each play's auto plays (one batch read per page; it
still pages in memory, as ADR 0140 allows for configuration-sized lists),
`play_update`'s `add_/update_/remove_auto_plays`, `workspace_update`'s
`auto_play_daily_cap`, `auto_play.*` events with live push and a web
follower, and the Access entries in the web and `client-core`. A doc play's
auto plays always run on `causer`.
