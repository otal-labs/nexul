# 01: Reading Nexul's logs out of OpenObserve

Pinned build: OpenObserve `v1.0.4` (`internal/install/openobserve.go`, `pinnedOpenObserve`),
the open-source build, tag commit `1e257e69ef0feb6029f3111d54697244a3f507db`. Source links
below are at that tag: `https://github.com/openobserve/openobserve/blob/v1.0.4/<path>`.
Docs links are the live openobserve.ai pages (not versioned); where docs and the tag's code
differ it says so.

## Answer

- One call does it: `POST {logs}/openobserve/api/default/_search` with Basic auth and a JSON
  body holding SQL, a microsecond time range, and `from`/`size`. The response has `hits`
  (rows), `total`, `took` (ms) and `scan_size`.
- Nexul's stream is `nexul` (the `stream-name` header, kept as is), org `default` (from the
  `/api/default` in the OTLP URL), type logs. A query names it as `FROM "nexul"`.
- Columns: level is `severity` (uppercase: `INFO`, `WARN`, `ERROR`, `DEBUG`), the message is
  `body`, `trace_id` and `source` are columns under their own names, and the OTLP service
  name is `service_name`. Server records carry no `source` column value; only browser ones
  have `source = 'web'`. Server vs web is best told apart by `service_name` / `source`.
- The token the server already holds (`NEXUL_OTLP_TOKEN`, which is `ZO_ROOT_USER_TOKEN`)
  cannot search. For a regular user, root included, the static token is accepted only on
  ingestion routes. A search needs the root email and password (`NEXUL_LOGS_PASSWORD`),
  which the server's own env does not get today.
- There is no read-only credential in this build. RBAC is enterprise-only. Creating a user
  through the API forces the Admin role, and the permission check always returns true. A
  service account can be created and its token rotated, which makes a separate, revocable
  secret, but it can do everything the root can.
- So the safety boundary has to be Nexul's: the server builds the SQL itself from a fixed
  column allowlist, after its own permission check. The API has no bind parameters, so
  values must be escaped by the server.
- Only the columns named in `SELECT` come back, so a secret in an attribute never leaves
  OpenObserve if the server never selects it. Never `SELECT *` (it returns every column).
- Result size: an explicit `size` / SQL `LIMIT` is honoured as given. With none, results are
  capped at 1000 rows (`ZO_QUERY_DEFAULT_LIMIT`). The query timeout is 600 s. There is no
  default cap on the time range. No published query-time figure for a small box exists; the
  response's `took` and `scan_size` are the measure, and the server should cap range and
  `size` itself.

## 1. The search API

Endpoint: `POST /api/{org_id}/_search`, under the base URI. Nexul sets `ZO_BASE_URI=/openobserve`
(`logsEnv`), so the server, calling OpenObserve directly on `127.0.0.1:NEXUL_LOGS_PORT`, posts to
`http://127.0.0.1:<port>/openobserve/api/default/_search`. Handler and the OpenAPI example:
[`src/api/search/src/search/mod.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/api/search/src/search/mod.rs)
(`search`, path `/{org_id}/_search`). Docs:
[Search API](https://openobserve.ai/docs/reference/api/search/search/).

Query-string options: `type` (`logs`, default), `is_ui_histogram`, `is_multi_stream_search`,
`validate` (checks queried fields exist in the schema and returns an error if not).

Request body (`SearchRequest`, [`src/config/src/meta/search.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/config/src/meta/search.rs)):

```json
{
  "query": {
    "sql": "SELECT _timestamp, severity, body FROM \"nexul\" WHERE severity = 'ERROR' ORDER BY _timestamp DESC",
    "start_time": 1760000000000000,
    "end_time": 1760086400000000,
    "from": 0,
    "size": 100
  },
  "use_cache": false,
  "timeout": 30
}
```

- `start_time` / `end_time`: int64 **microseconds** since the epoch. The docs say always send
  both; with no range the query scans everything.
- `from` (offset) and `size` (row count) page the result. Without an `ORDER BY`, a
  single-stream non-aggregate query is sorted `_timestamp DESC` (`src/search/src/sql/mod.rs`),
  so newest first.
- `timeout` (seconds) falls back to `ZO_QUERY_TIMEOUT`, default 600.
- Optional `agent_options.output_format` (`json`, `csv`, `md_table`) puts the rows in a
  string `data` field instead of `hits`; `agent_options.mode: "partition"` scans time
  partitions newest-first and stops early once enough rows are found. Both are in v1.0.4's
  `AgentOptions`.
- Response: `took` (ms), `hits` (array of row objects), `total`, `from`, `size`,
  `scan_size`, `scan_records`, `is_partial`, `function_error` (carries the "results are
  capped" warning), `trace_id` (OpenObserve's own query id, unrelated to Nexul's `trace_id`).
- A stream that does not exist yet fails with a "stream not found" error
  (`src/search/src/sql/mod.rs`, `SearchStreamNotFound`). The reader must treat that as "no
  logs yet".
- Paging by `from` re-runs the query each page. For a long tail, page by time instead:
  pass the last row's `_timestamp` as the next `end_time`.

## 2. Filtering, and how the stream name maps

**Stream.** The exporter sends header `stream-name: nexul` (`server/cmd/main.go`,
`internal/platform/logging/otlp.go`). The OTLP handler reads the header and runs it through
`format_stream_name` (illegal characters become `_`; upper case is lower-cased, since
`ZO_FORMAT_STREAM_NAME_TO_LOWERCASE` defaults to true), then writes to that stream in the URL's org
([`src/core/src/logs/otlp.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/core/src/logs/otlp.rs),
line 67; handler [`src/api/ingest/src/request/logs/ingest.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/api/ingest/src/request/logs/ingest.rs),
`otlp_logs_write`). `nexul` is already clean, so it is stored as `nexul`. A query names it
as the SQL table: `FROM "nexul"`. Without the header, OpenObserve would use `default`.

**Columns.** Each OTLP record becomes one flat JSON row (`src/core/src/logs/otlp.rs`,
lines ~137 to 215):

| Nexul side | Row column |
|---|---|
| slog message | `body` |
| slog level (`r.Level.String()` set as severity text by the OTel slog bridge) | `severity`: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| slog attribute `trace_id` (the UUIDv7 from `logging/trace.go`) | `trace_id` |
| slog attribute `source` (`"web"`, set in `logging/browser.go`) | `source` |
| slog attributes `user_id`, `user_agent`, `url` | same names |
| browser `attrs` map | nested, flattened with `_`: `attrs_stack`, `attrs_line` |
| resource attribute `service.name` (`nexul-server`) | `service_name` |
| record time | `_timestamp` (microseconds) |

Notes that matter for the reader:

- Keys are normalised by `format_key` (`src/config/src/utils/flatten.rs`): lower-cased, and
  anything not `[a-z0-9_]` becomes `_`. Nesting is flattened up to level 3 by default
  (`ZO_INGEST_FLATTEN_LEVEL`).
- Bridge conversion of slog values is in the bridge's own `handler.go`/`convert.go`
  (`go.opentelemetry.io/contrib/bridges/otelslog` v0.20.1, the version in `go.mod`): maps
  and groups become OTLP maps, durations become nanosecond integers.
- A caller attribute named `trace_id` is kept as a plain column only because Nexul sets no
  OTLP-level trace id. If a record ever carried a real OTLP trace id it would overwrite the
  attribute (the code sets `rec["trace_id"]` after copying attributes).
- Server records have no `source` attribute today. `source = 'web'` finds browser records;
  server records are `source IS NULL` or `service_name = 'nexul-server'`. The runners'
  and automations host's records will have to set their own `service.name`.

**Filter examples** (the server builds these; never take SQL from the caller):

```sql
-- level
WHERE severity IN ('WARN','ERROR')
-- a request or deploy trail by trace id (trace_id is a default secondary-index field)
WHERE trace_id = '0198...'
-- browser only
WHERE source = 'web'
-- message text; body is a default full-text field, so both work
WHERE match_all('deploy failed')
WHERE str_match(body, 'deploy failed')
WHERE body LIKE '%deploy failed%'
```

`body` is in the default full-text list and `trace_id` / `service_name` in the default
secondary-index list (`src/config/src/config.rs`, `_DEFAULT_SQL_FULL_TEXT_SEARCH_FIELDS`,
`_DEFAULT_SQL_SECONDARY_INDEX_SEARCH_FIELDS`, both on unless
`ZO_FEATURE_DEFAULT_INDEX_FIELDS_ENABLED=false`), so those filters are the cheap ones.
`match_all(...)` and `str_match(...)` are OpenObserve's own SQL functions (named in the
search endpoint's own description in `search/mod.rs`).

**Escaping.** `SearchRequest` has no parameter list; the SQL is one string. A value from an
agent (a message fragment, a trace id) has to be inserted with single quotes doubled, or
checked against a strict pattern (a trace id is a UUID). Level can be mapped from an enum.

## 3. Auth and roles

**What the server holds today.** `serverEnv` (`internal/install/install.go`) gives the server
`NEXUL_OTLP_USER` (= `NEXUL_LOGS_EMAIL`) and `NEXUL_OTLP_TOKEN` (= `NEXUL_LOGS_TOKEN`,
OpenObserve's `ZO_ROOT_USER_TOKEN`). It does **not** get `NEXUL_LOGS_PASSWORD`
(`logsEnv` hands that only to OpenObserve).

**The token cannot search.** In `validate_credentials`
([`src/api/common/src/auth/validator.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/api/common/src/auth/validator.rs),
line ~555), a non-service-account user's static token is accepted only when
`is_ingestion_allowed(method, path)` is true; the comment there calls it "an ingestion-only
credential" and ties it to a read-bypass advisory. On `/_search` it falls through to the
password comparison and fails. So the ingest credential Nexul already has is write-only by
design, and the server needs a new secret to read.

**Root can do everything.** Root email + password (HTTP Basic) passes. With the open-source
build, `check_permissions` always returns true
([`src/core/src/authz.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/core/src/authz.rs),
the `#[cfg(not(feature = "enterprise"))]` variant, line ~159). Root can read any stream,
delete data, and change settings.

**No read-only user in this build.**

- Docs say so outright:
  [RBAC](https://openobserve.ai/docs/user-guide/account-administration/identity-and-access-management/role-based-access-control/):
  "Not available in Open Source ... All users have unrestricted access to all features."
  The Viewer role is "read-only" only in Enterprise.
- Code, which is stricter than the docs: `create user` and `update user` overwrite the
  requested role with Admin when not built as enterprise
  ([`src/api/management/src/request/users/mod.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/api/management/src/request/users/mod.rs),
  lines ~198 to 200 and ~242 to 297). So a Viewer cannot even be created through the API,
  and custom roles are rejected ("Custom roles not allowed",
  [`src/core/src/users.rs`](https://github.com/openobserve/openobserve/blob/v1.0.4/src/core/src/users.rs),
  `post_user`).
- Per-stream permission checks (`check_stream_permissions`) are enterprise-only too
  (`#[cfg(feature = "enterprise")]` in the search handler).

**Service accounts exist in OSS, with full power.** `POST /api/{org}/service_accounts`
(`ZO_SERVICE_ACCOUNT_ENABLED`, default true) creates one and returns its token once. It
authenticates with Basic `email:token` on any route
(`validator.rs`, `user.role.is_service_account() && user.token.eq(&user_password)`, line
~489). Docs: [Service accounts](https://openobserve.ai/docs/user-guide/account-administration/identity-and-access-management/service-accounts/):
"Open-source: Service accounts have full access by default." The token can be rotated
(`PUT .../service_accounts/{email}?rotateToken=true`, Admin or Root only). What it buys over
the root password: a secret that can be revoked or rotated without changing the login, and
that is not the UI password. It does not narrow what the secret can do. Creating it needs
root credentials, so the install (which holds the password) would have to mint it and hand
it to the server.

**Practical choice for the server.** Either add `NEXUL_LOGS_PASSWORD` to `serverEnv`, or have
the install mint a service account and pass its email and token. Both give the server full
control of OpenObserve, so the Nexul permission check and the fixed-SQL builder are the only
read boundary. This is worth recording in the ADR that follows ADR 0008.

## 4. Limits and cost

- **Row cap.** If neither the SQL `LIMIT` nor `size` is positive, results are cut to
  `ZO_QUERY_DEFAULT_LIMIT` (default 1000) and a "results are capped" message is added
  (`src/search/src/utils.rs`, `is_default_query_limit_exceeded`;
  `src/search_service/src/cluster/http.rs`; `src/search/src/datafusion/optimizer/mod.rs`
  adds the limit plus 5 rows to detect overflow). A positive `size` or SQL `LIMIT` is applied
  as asked; I found no hard server-side maximum for it in the search handler. The
  separate "100 hits in summary mode" cap in the handler's doc text belongs to
  OpenObserve's MCP wrapper, not the HTTP API.
- **Time range.** No default cap: `ZO_DEFAULT_MAX_QUERY_RANGE_DAYS` defaults to 0
  (unlimited) and the per-stream `max_query_range` setting is unset. When one is set,
  OpenObserve clamps `start_time` and reports "Query duration is modified ..." in the
  response. Retention is `ZO_COMPACT_DATA_RETENTION_DAYS`, default 3650 days, so a day of
  logs is always present.
- **Timeout.** `ZO_QUERY_TIMEOUT`, default 600 s, per-request `timeout` overrides it.
- **Cost.** No figure for query time over a day on a small box is published. A time-bounded
  query on an indexed column (`trace_id`, `service_name`, `match_all` on `body`) scans only
  the matching files and reports `scan_size` and `took` in every response, so the first real
  measure comes from calling it on a live install. What the install does shape: Nexul caps
  OpenObserve's caches (`ZO_MEMORY_CACHE_MAX_SIZE=256`, `ZO_MEM_TABLE_MAX_SIZE=256`) and its
  unit `MemoryMax`, so large unbounded scans are the case to prevent. The server should
  enforce a maximum range (for example 24 hours), a maximum `size`, and a short `timeout`
  of its own. `agent_options.mode = "partition"` stops early on top-N reads over wide ranges.
- **Not a limit but a trap.** `use_cache` defaults to the instance's result-cache setting;
  send `"use_cache": false` for a log tail so a recent window is never served from cache.

## 5. Returning only some fields

Yes. Name the columns in `SELECT`; OpenObserve builds the result schema from the columns
the SQL uses (`src/search/src/sql/mod.rs`: `generate_schema_fields` for named columns versus
`generate_select_star_schema` for `*`). `SELECT *` returns every column in the stream
schema, including `user_agent`, `attrs_*` and anything added later, so it must not be used.

The server's reader should hold a fixed allowlist, for example
`_timestamp, severity, body, trace_id, service_name, source, url`, select only those, and add
attribute columns (`attrs_*`, `user_id`) only when a caller asks for them by name and is
allowed to see them. OpenObserve itself offers nothing finer (no per-column permission in
OSS), so what never leaves OpenObserve is exactly what the server never selects.

Browser records are redacted before they ship (`redact.Tokens` in
`internal/platform/logging/browser.go`), but server records are not filtered by that path,
so a `SELECT body` over server logs can still carry whatever the server logged. The
allowlist should therefore be per column and the response should go through the same
redaction before it reaches an agent.

## Where docs and the tag disagree

- Docs say Viewer can be assigned in open source and has no effect. At v1.0.4 the create and
  update handlers force Admin in the OSS build, so the role never reaches the database.
- The search docs page lists `timeout` and `search_type` and the `agent_options` fields; the
  `agent_options` fields exist at the tag (`AgentOptions` in `search.rs`), so they can be
  used on the pinned build, but a later bump should re-check them since they look new.
- The docs describe `scan_size` in MB; treat the unit as unverified until measured.
