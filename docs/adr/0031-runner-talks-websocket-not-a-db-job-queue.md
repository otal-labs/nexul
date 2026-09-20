# Runners take work over a WebSocket, not by polling a jobs table

The obvious shape for a SQLite-spined product is a `jobs` table the runners
poll, but SQLite has no `SELECT … FOR UPDATE SKIP LOCKED`, so two runners
polling the same table cannot safely claim different rows without a lock
dance that serializes them against the single writer. Runners instead hold a
long-lived WebSocket to the server, which pushes an assign frame to a chosen
idle runner and receives progress and results back on the same connection —
dispatch, cancellation, heartbeats, and liveness all fall out of the
connection being open, and the queue stays in memory on the server rather
than in the database.

Consequences: a job's queue position is not durable across a server restart,
and a runner that drops mid-job fails that job rather than having it
re-claimed by another runner (a second runner acting on the same machine
could corrupt its state).

Decided: 2026-07-24
