# 05: How a container dying after its deploy becomes an event

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

The runner looks at containers only while a deploy runs
(`internal/runner/observe.go`). After that, nothing notices one dying. What
watches, and what does the event carry?

- The signal: Docker's event stream (`die`, `oom`, restart counts) on each
  machine runner, filtered to containers Nexul deployed, or a periodic
  inspect. Also what happens across a runner restart or a lost connection,
  so a death in the gap is still reported.
- What counts as down: any exit, a non-zero exit, an OOM kill, or a restart
  loop past a threshold. How flapping is held to one event instead of a
  stream of them.
- The event: its topic and catalog row (ADR 0044), and its payload: stack,
  service, machine, exit code, OOM flag, restart count, and the last lines
  of output, masked as stack logs are. Those lines ride in the event and are
  never stored as logs (ADR 0091).
- A matching "came back" event, so a reaction can close what it opened
  (reverse states).
- How an auto play and an automation subscribe to it today, and whether
  either needs anything new to receive it. Which permission seeing the event
  takes (`stacks:logs` for the lines?).
- Whether a deploy's own stop and start, or a person stopping a stack, are
  told apart from a crash so they raise nothing.

Technical: answered in this ticket, not grilled with the owner.

## Answer

- **The runner watches Docker's event stream and the server decides.** Each machine runner (never
  a personal one) keeps `docker events --filter type=container --filter event=die --filter
  event=oom --filter event=start --format '{{json .}}'` running, restarted with backoff if it
  ends. It forwards each event as a new `container_event` frame: container name, compose project
  label, action, exit code, and time. The runner doesn't know which containers are Nexul's. The
  server matches events to the stacks on that machine by compose project (the stack slug) or run
  container name, and ignores the rest.
- **What counts as stopped:** a `die` with a non-zero exit code, or an `oom`. An exit 0 is a
  container that finished, such as a migration job, and raises nothing. A restart loop raises one
  stopped event at its first crash, and the episode stays open while it loops.
- **What doesn't count:** events for a stack while a deploy job for it is running on that runner,
  since a deploy recreates containers. The same goes for events for a stack being deleted.
- **Episodes are stored, so nothing repeats or goes missing.** A new forward-only migration adds
  `service_outages` (stack, service, container, started_at, exit_code, oom, restart_count,
  ended_at). A stopped event opens a row only if none is open for that service. On reconnect the
  runner sends a `container_states` snapshot (`docker ps -a` plus each container's exit code and
  `FinishedAt`). The server opens an outage for a Nexul container found exited non-zero with none
  open, and closes an open one whose container is running. So a death during a gap is reported
  once, and a server restart re-raises nothing.
- **"Came back" closes the episode.** A `start` closes the open outage only after the container
  has stayed running for 60 seconds, timed by the server from the start event. A die inside the
  window cancels the timer and keeps the episode open, so a restart loop never reads as recovered.
- **The events:** `service.container_stopped` and `service.container_recovered`, beside
  `service.updated` in the deploy domain's `Topics()`. Each carries workspace, project, stack,
  service, container, machine, outage id, exit code, OOM flag, restart count, and the times.
  Both go through the outbox and get `make event-schemas`. The audience is whoever reads the
  stack (`stacks:read` on its project), on live push and in automations, through the generated
  SDK topics.
- **The event carries no output lines.** This amends the map's charting note. Event payloads
  are stored (outbox, dead letters, webhook deliveries, see #694), so lines inside one would be
  stored container output, which ADR 0091 rules out. Docker keeps a crashed container's output
  until the container is removed, so the reader fetches it live: `stack_get` with `logs` for that
  service. The event names exactly which service to ask for.
- **Seeing it.** `stack_get` and the stack page show a service's open outage ("down since 10:42,
  exit 137, out of memory"), and the stack's services list it until it closes. How a person is told
  and how an agent is started without code is the next ticket.
