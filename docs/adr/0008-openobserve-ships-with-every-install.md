# OpenObserve ships in the compose stack and receives the server's logs over OTLP

Server logs only went to stderr, so a failed deploy left an agent with nothing
to query: the MCP server exposes the domain, not the logs. The fix is a log
store that (a) runs as one container inside a 4 GB host budget, (b) speaks
OpenTelemetry so C#, Go, and browser code all use the same SDK, and (c) has
an MCP server of its own so agents can search logs without Nexul
proxying them.

Measured on 2026-09-10 with the same 20,000-record burst: VictoriaLogs
11→39 MiB but logs only and no product UI; GreptimeDB 126→135 MiB but a
database with a query console, not a product, and an SQL-only MCP; OpenObserve
303→350 MiB, one Rust binary, logs + metrics + traces + session replay, MCP
built into the open-source edition and already speaking the 2026-07-28
stateless revision; the other multi-container candidate 535→648 MiB across
five containers; ClickStack 832 MiB→1.0 GiB in one container that bundles ClickHouse, MongoDB, and three
Node processes. OpenObserve is the only one that is one container *and* has
a real built-in MCP. It is AGPL-3, which is fine: it runs as a separate
process and is never linked into or vendored by Nexul.

Consequences: `docker-compose.yml` gains an `openobserve` service capped at
1 GiB (it otherwise sizes caches to half the host), the server gains
`NEXUL_OTLP_*` config and a fan-out `slog.Handler` so stderr keeps
working, and the OTLP auth header is built in the server because OpenObserve
accepts basic auth (email + root token) but rejects bearer tokens. Bundled
rather than opt-in by owner decision: a fresh install that cannot be
diagnosed is the problem being solved. Runner, automations, traces, and
the browser SDK follow later; this slice is server logs only.
