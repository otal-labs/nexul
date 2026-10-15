# 11: Build reading the instance's logs through Nexul

Type: task
Status: open
Blocked by: None — can start immediately

## What to build

Tickets 01 and 03, as decided: the server's OpenObserve reader (one `_search` call, server-built
escaped SQL over the fixed columns, Nexul's own range and row limits); the Logs instance area with
`logs:read`; `instance_get` with `logs`; `GET /api/instance/logs`, paged with a pinned `until`;
secret-named attributes dropped and tokens redacted; the upgrade re-rendering the server env with
the OpenObserve password.

## Surfaces

- MCP: `instance_get` gains `logs`; no new tool.
- HTTP: `GET /api/instance/logs`, in the route permission file.
- Permission: `logs:read` in the catalog, `make permission-catalog`, the Access card's Logs area.
- Docs: the logs guide rewritten around the Nexul tool; a successor to ADR 0008; `CONTEXT.md`
  names Nexul's own logs.
