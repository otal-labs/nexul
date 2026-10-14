# 02: How a server runner's and the automations host's logs reach OpenObserve

Type: grilling
Status: open
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
