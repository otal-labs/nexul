# 05: How a container dying after its deploy becomes an event

Type: grilling
Status: open
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
