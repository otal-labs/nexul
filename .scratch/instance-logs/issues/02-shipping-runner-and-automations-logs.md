# 02: How a server runner's and the automations host's logs reach OpenObserve

Type: grilling
Status: resolved
Blocked by: None — can start immediately

## Question

OpenObserve listens on `127.0.0.1` on the server host, and only the server
ships to it (`logging.NewWithOTLP` in `server/cmd/main.go`). A runner calls
`logging.New` and writes only to its own journal. The automations host
(`automations/`, TypeScript) reaches the server only through
`NEXUL_SERVER_URL` and its credential file. How do their logs get there?

- The road: over the connection each already holds to the server, with the
  server re-emitting the lines to OpenObserve the way `logging.BrowserHandler`
  does for the browser; or straight OTLP to the public `/openobserve/` proxy,
  which would need OpenObserve credentials on every remote machine; or
  something else.
- Which runners ship. A runner serving a machine ships. A personal runner on
  someone's computer never does (decided while charting). How does a runner
  know which one it is?
- What each line carries, so a reader can tell where it came from: `source`
  (`runner`, `automations`), the machine or host name, the runner version.
- What happens while the server is unreachable: drop, a bounded buffer, or
  replay on reconnect. The browser relay drops after its bounded queue.
- Levels: everything at the configured level, or warn and error only like
  the browser.
- What the runner's own panic containment (`internal/runner`) and the
  automations host's sandbox errors log, and whether that is enough to debug
  from.

Technical: answered in this ticket, not grilled with the owner.

## Answer

Both relay through the server over the connection they already hold. No OpenObserve credential
leaves the server host, and nothing new has to listen beyond `127.0.0.1`.

- **Runner: a `log` frame.** A new runner → server frame carries a batch of records (time,
  level, message, attributes). The runner's logger becomes a fan-out: its journal as today, plus
  a bounded in-memory queue the client flushes as `log` frames every 2s while connected. The
  instance runner on the server host takes the same road; there is no special case.
- **Automations host: `POST /api/automation-hosts/self/logs`.** It sits beside `self/assignments`,
  authenticated by the host credential. `log()` in `automations/src/log.ts` keeps writing its JSON
  line and also queues the record, flushed every 2s. Only the host's own lines ship. A script's
  output stays where automation runs keep it, because that is the workspace's output, not Nexul's.
- **Only machine runners ship.** A personal runner never attaches the queue (`cfg.Mode`), and the
  server drops a `log` frame from a personal connection (`runnerConn.personal`) without reading it.
- **The server stamps where a line came from; the sender can't.** Each line is re-emitted with
  `source` (`runner` or `automations`), the runner or host name and id, the machine (the runner's
  `machineID`, the host's `Machine` since #682), and the sender's version, all taken from the
  authenticated connection. Any record attribute with one of those names is dropped. Batches and
  message lengths are capped like the browser's, and messages and string attributes pass through
  `redact.Tokens`.
- **Relayed lines go to OpenObserve only, not the server's journal.** The sender's own journal
  already holds them. `logging` exposes its OTLP handler so the relay writes to it alone. The
  browser relay keeps writing to both for now.
- **While the server is unreachable:** keep the newest 1000 records and send them on reconnect,
  since why the connection dropped is often the line worth reading. On overflow the oldest go,
  and one `dropped N log lines` warning is sent with the next batch. A restart of the sender loses
  its queue; its journal still has those lines.
- **Levels:** the sender's configured level (info by default). Runner and host logs are low
  volume, and build or deploy output already travels in `deploy_log` frames, not slog.
- **Enough to debug from:** the runner's panic containment (#578) and the host's sandbox errors
  already log through these loggers, so they ship with no extra work. Any gap found becomes a
  build follow-up in the walkthrough.
