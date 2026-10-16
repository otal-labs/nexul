# 11: Build reading the instance's logs through Nexul

Type: task
Status: resolved
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

## Built

- The reader makes three calls, not one: the stream's schema first (OpenObserve refuses a query
  naming a column the stream lacks, and `attrs` needs the column names), then `count(*)` for
  `total` (its own `total` is the page size), then the page. Secret-named columns are never
  selected.
- Values holding a backslash or a control character are refused, since OpenObserve reads a
  backslash as an escape and doubled quotes alone let an injection through.
- `GET /api/instance/logs` is gated in the use-case and by `Scope(permissions.LogsRead)`; the
  route-permission file lists writing routes only, so it isn't there.
- Runner and automations-host lines need ticket 12 before those sources return anything.
