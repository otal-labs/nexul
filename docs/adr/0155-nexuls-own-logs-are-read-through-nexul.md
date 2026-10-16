# Nexul's own logs are read through Nexul

ADR 0008 put the logs in OpenObserve and left reading them to OpenObserve's own UI and MCP. Both need the root
password from the install's `.env`, know nothing of Nexul's people, and hand anyone who holds the password every
stream, setting and delete in the log store. An agent debugging an instance with no shell on its box had no road that
Nexul's permissions governed.

Decision: the server reads the log store on the caller's behalf, after its own permission check.

- **Why the server stands in front.** OpenObserve's open-source build has no read-only user: role-based access control
  is enterprise-only, creating a user forces the Admin role, and a service account has full power. The token the server
  sends logs with is an ingest credential and cannot search. Whatever login the server reads with can do everything,
  so Nexul's check and the query it builds are the only boundary.
- **The query is the server's.** One search call with the log store's root login (`NEXUL_LOGS_EMAIL`,
  `NEXUL_LOGS_PASSWORD`, which `nexul install` and `nexul upgrade` hand the server unit). The SQL names columns taken
  from the stream's own schema, never `*`, and leaves out a column whose name says it holds a secret, so those never
  leave OpenObserve. Every value is quoted, because the API has no bind parameters, and a value holding a backslash
  is refused, because OpenObserve reads a backslash as an escape. Nexul sets what OpenObserve does not: a range of at
  most 7 days, 1 to 500 lines, a 30 second timeout, and no result cache. Lines come back with tokens redacted and cut at
  2000 characters.
- **The surface.** `logs:read`, a new instance area handed out like the others; `instance_get` takes a `logs` filter
  and `GET /api/instance/logs` takes the same as a query string. The list shape of ADR 0140, with the `until` a first
  page pins so new lines do not shift later pages.
- **When it cannot answer.** An instance with no log store refuses with that reason, and a log store that does not
  answer is a conflict that says to retry. Neither is ever an empty list.

What stays: the server still ships its own logs to OpenObserve over OTLP (ADR 0008), the Owner still has the
OpenObserve UI and MCP behind the root password, and a stack's container output stays behind `stacks:logs`
(ADR 0091).

Trade-offs: the server holds a login that can delete logs, which only the install and the server's own code ever
use, and a read costs three calls to OpenObserve (the stream's columns, a count, a page) because its `total` is the
page's size and `SELECT *` would defeat the secret filter. Rejected: an OpenObserve service account, which narrows
nothing; a second OpenObserve in front with enterprise roles, which is not open source; and returning only
the headline fields, which hides the error and ids a server line carries, the reason a reader asks.

Succeeds the reading half of ADR 0008. Decided 2026-10-16.
