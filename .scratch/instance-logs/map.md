# Wayfinder map: debugging an instance without its box

Charted 2026-10-14 with the owner. When an install misbehaves, finding out
why today means a shell on the box. The server's logs and the browser's
warnings and errors reach OpenObserve, but reading them takes the shared root
password from the install's `.env`. Runners and the automations host keep
their logs only in their own machine's journal. Containers can already be read
live through the runner: a stack's through `stack_get` with `logs`, and a
computer's through `computer_list` with `container_logs`. Nothing notices a
container dying after its deploy, though, so no agent is woken to look. And
a bug seen in the phone app or the desktop launcher has no road to the repo.

## Destination

Built and merged. An agent, or a person, holding the right permission can
read the instance's own logs through Nexul, with no box and no root password:
the server, the browser, server runners and the automations host. A personal
runner's logs and its computer's containers can be read by its owner and by
anyone the owner's share allows. A container that dies after its deploy
raises an event an auto play or an automation can act on. A triage playbook
lets a person's own agent turn a bug into an issue on `otal-labs/nexul`.
This map carries the build: build tickets are worked here once the decision
tickets they wait on are resolved.

## Notes

- Decided while charting:
  - **Logs is a new instance area.** It is handed out like the others: the
    instance Owner holds it by right, and its Access card gives it to a role
    of any workspace or to a person. It reads Nexul's own logs, not
    containers' (`stacks:logs` keeps those).
  - **Agents go through Nexul, not OpenObserve's own MCP.** That MCP needs
    the root password and knows nothing of Nexul's people. The server reads
    OpenObserve on the caller's behalf, after the permission check.
  - **Server runners and the automations host ship their logs** to
    OpenObserve, as the server does.
  - **A personal runner keeps its logs on its computer and never ships
    them.** They are read live through the runner, like container logs, by
    the computer's owner and by anyone whose share on that computer allows
    it. One "Read logs" action on a computer's share covers both the runner's
    own logs and the computer's containers' logs. It is off by default.
  - **Container output stays live and is never stored** (ADR 0091). A
    "container stopped" event carries no output lines either, because event
    payloads are stored; the reader fetches them live (ticket 05).
  - **Container-down is an event.** The runner notices a deployed container
    dying after its deploy, and the server publishes an event that auto plays
    and automations can react to.
  - **Triage starts as a playbook over MCP.** A skill plus whatever tool
    support it needs hands the person's own agent the facts. The agent
    diagnoses against the source at the installed version and opens an issue
    on `otal-labs/nexul`.
  - **No browser page for Nexul's own logs for now.** The Owner keeps
    OpenObserve's UI. Retention stays at what OpenObserve keeps today.
- Technical questions are answered in their ticket, not grilled. The triage
  ticket is a grilling ticket: what a public issue may carry is the owner's
  call.
- Read before code: `practices/go.md`, `practices/architecture.md` and
  `practices/mcp.md` for the server and runner; `practices/typescript.md`
  for the automations host. Tools are a budget (`practices/mcp.md`, section
  4): extend an existing tool before adding one.
- Hit every surface. Each build ticket names its MCP tool, HTTP route,
  permission, event catalog row, and guide page. The logs guide
  (`website/src/content/docs/docs/guide/logs.md`) is rewritten around the
  Nexul tool, and ADR 0008 gets a successor if the root-password road stops
  being the agent's road.

## Decisions so far

- [How the server reads the logs OpenObserve holds](issues/01-reading-openobserve-from-the-server.md): one `_search` call with root credentials (no read-only user exists), server-built escaped SQL over a fixed column list, Nexul's own limits on range and rows.
- [How a server runner's and the automations host's logs reach OpenObserve](issues/02-shipping-runner-and-automations-logs.md): relayed through the server, a `log` frame from machine runners and `self/logs` from the host, with the source stamped by the server and a bounded queue of 1000 across disconnects; personal runners never ship.
- [The Logs instance area and how an agent reads the instance's logs](issues/03-the-logs-tool-and-area.md): `logs:read` in a new Logs area; `instance_get` takes `logs` (no new tool) and `GET /api/instance/logs`, paged with a pinned `until`; the server filters secret-named attributes and redacts tokens; upgrade re-renders the server env to carry the OpenObserve password.
- [How a personal runner keeps and serves its own logs, and the share's Read logs](issues/04-personal-runner-logs-and-the-share.md): its own rotated `runner.log` on every OS, tail from the file and follow from the logger, `logs_request` with `target: runner`; `computers:read_logs` on a share covers runner and container logs, masked for sharees.
- [How a container dying after its deploy becomes an event](issues/05-container-stopped-event.md): machine runners forward Docker's die/oom/start events and a reconnect snapshot; the server matches them to stacks, counts non-zero exits and OOM outside deploys, stores outages, and publishes `service.container_stopped` and `service.container_recovered` (60s steady) with no output lines.
- [How a stopped container reaches an agent without writing an automation](issues/07-from-stopped-container-to-agent.md): a stack switch opens one crash ticket per service; stack plays become a third play type with crashed and came-back moments; any play can run on a chosen computer; a shipped "Investigate a crash" play never deploys; everyone who reads a stack watches it unless they turn Watching off.

## Not yet specified

- The build slices for each decision, and the walkthrough on a real install
  with no shell on the box. They graduate as the decision tickets resolve.
- Whether `CONTEXT.md` needs its own term for Nexul's own logs. Its
  "Container logs" entry says to avoid "service logs (Nexul's own, which
  OpenObserve holds)" but gives them no name of their own.

## Out of scope

- A browser page for reading Nexul's own logs. Ruled out for now; agents are
  the reader and the Owner has OpenObserve.
- A `nexul triage` command on the server host. It comes after the MCP
  playbook, as its own effort.
- Shipping logs from the phone app or the desktop launcher. Triage covers
  them.
- Storing container output anywhere (ADR 0091).
- Changing how long OpenObserve keeps logs.
