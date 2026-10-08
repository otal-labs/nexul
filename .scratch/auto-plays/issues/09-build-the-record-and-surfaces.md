# 09: Build the auto play record, permissions, HTTP and MCP

Type: task
Status: open
Blocked by: 02

## Question

Build what 02 decided: the migration (forward-only, production data is
live), the repo and use cases with the `autoplays` permission checks, the
HTTP routes, the MCP tool changes, events and live push, the Access page
entry, and Go tests including the error paths.

## Notes

- Backfill per 02's answer: `autoplays:read` from `plays:read`, `read`
  and `write` from `plays:write`, `delete` from `plays:delete`.
