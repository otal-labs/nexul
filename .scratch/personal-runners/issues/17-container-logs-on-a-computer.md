# 17 — Container logs on a computer with Docker

**Status:** ready-for-agent

**Blocked by:** 05, 06

Read first: `practices/go.md`, `practices/react-guide.md`, `practices/mcp.md` (section 7), ADR 0091, the
spec (Later: container logs on a computer).

## What to build

- Facts gain `docker: {available, version}`.
- A personal runner with Docker accepts `discover` and `logs_request`; the server adds a container list
  and `OpenComputerLogs` keyed by computer, owner-only.
- The computer row's Containers section with the stack page's log viewer (snapshot and live tail).
- `computer_list` with `id` takes `container_logs {container, tail}`.

## Acceptance criteria

- [ ] ADR 0091's limits hold unchanged (16 streams, 1000-line tail, 64 KB a second).
- [ ] `TestComputerContainers_OnlyTheOwnerSeesThem` over HTTP, MCP and the logs socket.
- [ ] A computer without Docker shows no Containers section and accepts no `logs_request`.
