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

## Decisions so far

## Not yet specified

- Whether automations can subscribe to container output (an alert on a log
  line). Out of reach until the source of truth for logs is decided.

## Out of scope

- Shelling into a container (exec). A separate effort if it is ever wanted.
- Logs of Nexul's own services; OpenObserve already covers those.
