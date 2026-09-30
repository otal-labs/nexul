# 03 — How the runner serves logs on demand

**Type:** task
**Status:** resolved
**Blocked by:** 01

## Question

The server can only ask the runner for things over the runner's one WebSocket, and today nothing streams back on request. Decide the frames (request, chunk, end, cancel), the Docker command (`docker logs --timestamps --tail N [--follow]` or the collector from ticket 01), limits (lines per request, bytes per second, idle timeout), what happens when two people watch the same container, and how a closed browser tab stops the stream on the host.

## Answer

Decided 2026-09-30.

- **Frames.** Four new frames on the runner WebSocket:
  - `logs_request {id, container, tail, follow}` from the server;
  - `logs_chunk {id, lines: [{ts, stream, line}]}` from the runner;
  - `logs_end {id, error}` from the runner;
  - `logs_cancel {id}` from the server.
- **Command.** `docker logs --timestamps --tail <n> [--follow] <container>`,
  with stdout and stderr read from separate pipes so each line knows its
  stream.
- **Batching** works like the deploy log: a flush every 250ms or at 32KB.
- **Limits.** `tail` defaults to 200 and is capped at 1000. A stream sends at
  most 64KB/s; beyond that, lines are dropped and one line says how many
  were skipped. A runner holds at most 16 open streams, and one more is
  refused with a readable error.
- **One stream per viewer.** Two people watching the same container run two
  `docker logs` processes. There is no fan-out, because the process is
  cheap and fan-out would need a snapshot and join protocol.
- **Lifecycle.**
  - The server sends `logs_cancel` when the viewer leaves, and the runner
    kills the process.
  - A runner disconnect ends every stream.
  - A container that is not running still returns its last lines with
    `follow` ignored, because Docker keeps the log of a stopped container
    until it is removed.
- **The server masks** (ticket 02), because it holds the env values.
