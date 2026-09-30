# Container logs are read live from Docker, not stored

A running container's own output was only reachable with a shell on its host. There were two places to read it
from: Docker's own log (`docker logs`), which needs nothing new on a machine but goes with the container and has no
search, or the install's OpenObserve (ADR 0008), which would give search and retention at the cost of a collector on
every runner host and disk on the server.

Decision: Nexul reads a service's container logs live from Docker through the runner holding the machine's WebSocket
(ADR 0031), and stores none of it.

- The server sends `logs_request {id, container, tail, follow}`. The runner runs
  `docker logs --timestamps --tail <n> [--follow] <container>` with stdout and stderr on separate pipes, sends the
  lines as `logs_chunk {id, lines: [{ts, stream, line}]}` batched like the deploy log (250ms or 32KB), and closes with
  `logs_end {id, error}`. `logs_cancel {id}` kills the process, and a runner disconnect ends every stream.
- One stream per viewer: two people watching one container run two processes, because fan-out would need a snapshot
  and join protocol and the process is cheap.
- Limits: a tail is capped at 1000 lines, a following stream at 64KB a second, and a runner at 16 open streams, the
  next one refused with a readable error. Over the rate, lines are dropped and one "N lines skipped" line takes their
  place; a snapshot is bounded by its tail and is never rate-capped.
- A snapshot route returns the last lines. A live tail is a WebSocket per viewer whose lifetime is the stream's, not a
  topic on `/ws/events`, which broadcasts to every socket and has no per-viewer start and stop. A viewer whose socket
  falls behind gets a skipped line instead of a growing buffer on the server.
- Reading takes `stacks:logs` (ADR 0057), because output carries secrets. Every role holding `stacks:write` received
  it on upgrade. Before a line leaves the server, every value of the stack's own env that is six characters or longer
  becomes `••••`.

The trade-offs: the output of a container that a redeploy replaced or removed goes with it; there is no search across
containers, restarts, or redeploys; nothing is readable while the machine's runner is offline; and masking covers only
the values Nexul holds, so a secret the app derives or fetches itself is shown.

Searchable history, shipping container output into the install's OpenObserve, is built when someone needs to search
across restarts or redeploys, or an automation needs to react to a log line.

Decided 2026-09-30.
