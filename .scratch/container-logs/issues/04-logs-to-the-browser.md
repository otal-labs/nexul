# 04 — How logs reach the browser and the phone

**Type:** task
**Status:** resolved
**Blocked by:** 01, 03

## Question

Decide the HTTP route for a snapshot and the live path for a tail: a topic on the existing `/ws/events` hub with a `live_audience` rule (the agent-stream pattern, never stored), or a dedicated socket per viewer. Covers reconnect, back-pressure on a chatty container, and keeping the route off the setup-pass allowlist.

## Answer

Decided 2026-09-30.

- **Snapshot:** `GET /api/stacks/{id}/services/{name}/logs?tail=N` returns
  the last N lines without following. It is used for "copy all" and by the
  MCP tool.
- **Live:** a dedicated WebSocket per viewer at
  `/ws/stacks/{id}/services/{name}/logs?tail=N`.
  - Opening the socket starts a following stream; closing it sends
    `logs_cancel`. The socket's lifetime is the stream's lifetime.
  - It is not a topic on `/ws/events`: that hub broadcasts to every socket
    and has no per-viewer start and stop.
  - Auth is the same as `/ws/events`: the session cookie in a browser, and
    `?token=` from the phone.
  - Lines are never stored.
- **Back-pressure.** When a viewer's socket falls behind, lines are dropped
  and one "N lines skipped" line takes their place, as on the runner.
- **Reconnect.** The client reopens with a small tail, and duplicates across
  the gap are accepted.
- **Access.** `stacks:logs` is checked at open and against the stack's
  project (ADR 0087). Neither route joins the setup-pass allowlist.
