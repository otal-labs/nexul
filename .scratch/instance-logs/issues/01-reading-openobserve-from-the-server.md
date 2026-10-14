# 01: How the server reads the logs OpenObserve holds

Type: research
Status: resolved
Blocked by: None — can start immediately

## Question

What does the pinned OpenObserve build (the version `internal/install` puts
on the box) offer for a server-side reader that answers an agent's question
about Nexul's own logs?

- The search API: its endpoint, request shape (SQL or otherwise), time range,
  limit and paging, and response shape.
- How to filter by the attributes Nexul's logs carry today: `source`
  (`server`, `web`), level, `trace_id`, message text. Also how the stream
  name the OTLP exporter sets (`internal/platform/logging/otlp.go`) maps to
  what a query names.
- Auth: whether the OSS build can mint a read-only user or service account,
  so the server need not query as root. If not, what the root credentials
  the server already holds (`NEXUL_OTLP_USER`, `NEXUL_OTLP_TOKEN`) can do.
- Limits and costs: query time over a day of logs on a small box, and any
  server-side caps on result size.
- Whether a query can return only the fields asked for, so secrets in an
  attribute an agent didn't ask for never leave OpenObserve.

Cite the OpenObserve source or docs for the pinned version for each answer.
Findings go in `research/01-openobserve-search.md`.

## Answer

Findings with sources: [research/01-openobserve-search.md](../research/01-openobserve-search.md),
read at the pinned build, OpenObserve v1.0.4 (open source).

- One call: `POST /api/default/_search` with Basic auth, SQL over the `nexul` stream, a
  microsecond time range, and `from`/`size`. The level is `severity`, the message `body`;
  only browser lines carry `source`, so server lines are told apart by `service_name`.
- The ingest token the server holds cannot search. The server needs the root email and
  password, which `serverEnv` does not pass it today.
- This build has no read-only user: RBAC is enterprise only and every user is an Admin.
  Nexul's own permission check is the only boundary, so the server builds the SQL itself,
  escapes every value (the API has no bind parameters), and selects a fixed column list,
  never `*`.
- Nothing caps the time range or `size` on Nexul's install, so the server sets its own
  limits on range, rows and timeout, and sends `use_cache: false` for a tail.
