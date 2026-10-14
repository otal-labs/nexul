# 03: The Logs instance area and how an agent reads the instance's logs

Type: grilling
Status: resolved
Blocked by: 01

## Question

Logs becomes an instance area: the instance Owner holds it by right, and its
Access card hands it to a role or a person (`instanceAreas` in
`internal/platform/permissions/permissions.go`). What exactly does it gate,
and what does an agent call?

- The action or actions (`logs:read` alone?) and their label in the catalog.
- The MCP surface, within the tool budget (`practices/mcp.md`, section 4):
  a `logs` query on `instance_get`, beside how `stack_get` takes `logs`, or a
  new tool that earns its place. Name it and its parameters: source, level,
  time range, `trace_id`, text, limit.
- The HTTP route the browser and integrations reach, even with no page for
  now (ADR 0019), and the gateway scope it needs.
- What comes back: the fields per line, the cap on lines and bytes, and how
  paging works over the search API ticket 01 describes.
- Redaction before a line leaves the server: tokens and secrets in messages
  and attributes. Browser logs are already redacted (`web/src/lib/browserLogs.ts`);
  server attributes are not. Who holds Logs matters here, because it need
  not be the Owner.
- The answer when OpenObserve is down or not installed: an error that says
  so, not an empty list.
- What the logs guide says afterwards. Does OpenObserve's own MCP stay
  documented as the Owner's road, or drop out of the guide?

Technical: answered in this ticket, not grilled with the owner.

## Answer

- **The area and its action.** Add one action, `logs:read` ("Read the instance's logs"), as a
  `logs` domain row with `read` only in `AreaInstance`, beside `audit`. Add an instance area,
  `{"logs", "the instance's logs", [LogsRead]}`. The Access card, the Owner-by-right rule, the
  generated client catalog and the route-permission table then come with it. It covers every
  source the store holds: server, web, runner and automations. It never reaches a container's
  output, which stays behind `stacks:logs` and a computer's share.
- **MCP: `instance_get` takes `logs`, no new tool.** This is the same shape as `stack_get`'s `logs`,
  so the tool budget is unchanged. `instance_get` keeps needing `instance:read`; passing `logs`
  also needs `logs:read`, refused with the area's reason when missing. Parameters, all optional:
  - `source`: `server`, `web`, `runner`, `automations`; all when omitted.
  - `level`: the lowest level returned; defaults to `info`.
  - `since` (a duration like `30m`, or a time) and `until`: default the last hour, at most 7 days
    apart.
  - `trace_id`, plus `name` (a runner or host name) and `text` (a substring of the message).
  - `limit` (1 to 500, default 100) and `offset`.
  Lines are newest first. The answer echoes the `until` it used: an `until` left out is pinned
  to now on the first page, and the caller passes it back with `next_offset`, so new lines don't
  shift the pages. The shape is the list shape of ADR 0140: items, total, has_more, next_offset.
- **HTTP: `GET /api/instance/logs`**, with the same parameters as query strings, gated by
  `logs:read` and given `Scope(permissions.LogsRead)` in the integrations gateway. It has no page
  for now (ADR 0019).
- **What a line carries:** `time`, `level`, `source`, `message`, `trace_id`, and `name`,
  `machine` and `version` for relayed lines. For web lines it adds `user_id` and `url`. Every
  other column is gathered into `attrs`, because the error, stack and ids a server line carries
  are what a reader came for. OpenObserve's own bookkeeping columns (`_timestamp`, `span_id`,
  `severity_number`, `flags`, `dropped_attributes_count`) are dropped.
- **Redaction happens in the server before anything is returned.** The server builds the SQL
  itself and escapes every value (research 01). It drops any attribute whose name holds
  `token`, `secret`, `password`, `key`, `auth`, `cookie`, `credential` or `env`. Every string
  value passes through `redact.Tokens`, and messages and attribute values are cut at 2000
  characters. The server is the one place values are filtered, so a non-Owner holding `logs:read`
  sees exactly what the filter lets out.
- **Credentials.** `serverEnv` passes `NEXUL_LOGS_EMAIL` and `NEXUL_LOGS_PASSWORD`, both already in
  the install's `.env`. `upgradeUnit` keeps a unit's saved env today, so an upgrade must re-render
  the server unit's env from `serverEnv` too. Otherwise existing installs never get the
  credentials. Search runs with Basic auth, a 30s timeout and `use_cache: false`.
- **When the store can't answer.** With no `NEXUL_LOGS_URL` (a dev server), it refuses with "this
  instance keeps no logs". When OpenObserve doesn't answer, it gives a conflict naming the
  log store and suggesting a retry. Never an empty list.
- **The guide.** `guide/logs.md` leads with `instance_get` and its `logs` parameter, the
  Logs area, and how the Owner hands it out. OpenObserve's UI and its MCP stay documented as
  the Owner's own road, since they need the root password. A new ADR succeeds the reading half of
  ADR 0008: "Nexul's own logs are read through Nexul". It records why the server stands in front
  of OpenObserve, namely that OpenObserve has no read-only user in its open-source build.
