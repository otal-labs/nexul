# 12: Build shipping server runner and automations host logs

Type: task
Status: resolved
Blocked by: 11

## What to build

Ticket 02, as decided: a `log` frame from machine runners and `self/logs` from the automations
host, relayed by the server into OpenObserve with the source stamped by the server, and a bounded
queue of 1000 across disconnects. Personal runners never ship.

## Surfaces

- Runner and host protocol: the new frame and route, with their tests.
- Reading them goes through ticket 11's tool; filters for the source are documented there.
- Docs: the logs guide lists the sources.

## Built

- Runners send a `log` frame (at most 50 records, 48 KiB); the host posts to
  `/api/automation-hosts/self/logs`. Both flush every 2s from a 1000-record queue that drops the
  oldest and leads the next batch with a "dropped N log lines" warning.
- The server stamps `source`, `name`, `source_id`, `machine` (the machine's name, so it filters
  alike for runners and hosts) and `version`, drops payload attributes with those names, redacts,
  and writes to the log store only, not its own journal.
- A script's own output stays on the host; only the host's records ship.
