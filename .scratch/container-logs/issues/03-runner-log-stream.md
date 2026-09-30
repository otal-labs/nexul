# 03 — How the runner serves logs on demand

**Type:** task
**Status:** open
**Blocked by:** 01

## Question

The server can only ask the runner for things over the runner's one WebSocket, and today nothing streams back on request. Decide the frames (request, chunk, end, cancel), the Docker command (`docker logs --timestamps --tail N [--follow]` or the collector from ticket 01), limits (lines per request, bytes per second, idle timeout), what happens when two people watch the same container, and how a closed browser tab stops the stream on the host.
