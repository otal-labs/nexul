# 12: Build shipping server runner and automations host logs

Type: task
Status: open
Blocked by: 11

## What to build

Ticket 02, as decided: a `log` frame from machine runners and `self/logs` from the automations
host, relayed by the server into OpenObserve with the source stamped by the server, and a bounded
queue of 1000 across disconnects. Personal runners never ship.

## Surfaces

- Runner and host protocol: the new frame and route, with their tests.
- Reading them goes through ticket 11's tool; filters for the source are documented there.
- Docs: the logs guide lists the sources.
