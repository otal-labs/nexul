# Wayfinder map: Container logs

**Label:** wayfinder:map

## Destination

A locked design, ready to slice into implementation tickets, for reading what
a deployed service's container prints (stdout and stderr) from inside Nexul:
the stack page and the topology canvas in the browser, agents through MCP, and
the phone app. Today the only log anywhere is the deploy log; a running
container's own output needs a shell on the host.

## Notes

- Domains: `internal/runner` (the runner WebSocket and its frames),
  `internal/deploy` (stacks, containers, the deploy log path),
  `server/cmd/live_audience.go` (live push rules), `internal/platform/permissions`,
  `web/src/components/stack`, `web/src/components/topology`, and `native/`.
- What exists to build on:
  - The runner holds one WebSocket to the server. `discover` is the only
    server-to-runner request with a reply, and nothing streams back on demand
    (`internal/runner/protocol.go`, `handler.go`).
  - The deploy log is stored as rows (`deploy_log_lines`) and the browser
    refetches the whole log on `deploy.updated` (PR #35). Agent stream frames
    are the pattern for an ephemeral stream that is never stored
    (`live_audience.go`, `useAgentStreamStore`).
  - Containers are recorded per stack with their Docker name (`ContainersTable`
    on the stack page, `StackScreen` on the phone); the machine lives on the
    stack.
  - OpenObserve ships with every install (ADR 0008) and holds the server's own
    logs; nothing ships container output to it today.
  - The MCP surface sits at its ceiling (ADR 0081), and `logs` is not an
    allowed verb, so the capability extends an existing tool unless an ADR
    raises the budget.
  - Permissions: `stacks` is read/write/delete and a domain may declare its own
    verb (ADR 0057).
  - Container output can hold secrets (env values the app prints, tokens in
    stack traces); nothing redacts the deploy log today.
- Skills every session should consult: `/grilling` and `/domain-modeling` for
  grilling tickets, `design-mode` for the look ticket.
- Read `practices/` per `AGENTS.md` before any code, including prototypes.
- Standing preferences from the owner, treat as fixed:
  - Hit every surface: web, HTTP, MCP, live push, permissions, the phone.
  - Signals over gates.
  - `web/` is built at 768px first; the phone app covers phones.
  - No "v1" or version-tier framing.

## Build tickets

Every decision was settled on 2026-09-30 and sliced into tickets 08 to 11:
the backend (runner frames, `stacks:logs`, the snapshot and live routes, the
ADR), `stack_get` logs, the web Logs view, and the phone screen.

## Decisions so far

- [01 — What a container's logs show](issues/01-what-logs-show.md) — a live tail read from Docker, timestamped, stderr marked, nothing stored; searchable history deferred.
- [02 — Who may read container logs](issues/02-who-reads-logs.md) — its own verb `stacks:logs`, on by default wherever `stacks:write` is; the stack's env values masked.
- [03 — How the runner serves logs on demand](issues/03-runner-log-stream.md) — request, chunk, end and cancel frames over `docker logs`; one stream per viewer, rate and count caps.
- [04 — How logs reach the browser and the phone](issues/04-logs-to-the-browser.md) — a snapshot route plus a dedicated WebSocket per viewer whose lifetime is the stream.
- [05 — Container logs through MCP](issues/05-logs-for-agents.md) — an optional `logs` argument on `stack_get`, bounded, masked, gated by `stacks:logs`.
- [06 — Where logs open and how they look](issues/06-logs-look.md) — a Logs view in the stack page with service tabs, one terminal-style block, All / Errors, Pause, Copy, Download, a Live marker.
- [07 — Container logs on the phone](issues/07-logs-on-the-phone.md) — tap a service row for a read-only following view.

## Not yet specified

- Whether automations can subscribe to container output (an alert on a log
  line). It needs stored logs, so it waits for searchable history (ticket 01).

## Out of scope

- Shelling into a container (exec). A separate effort if it is ever wanted.
- Logs of Nexul's own services; OpenObserve already covers those.
