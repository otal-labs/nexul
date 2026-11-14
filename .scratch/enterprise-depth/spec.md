# Enterprise depth: SAML beside OIDC, and audit log retention and streaming

**Status:** needs-triage

Two things the enterprise edition lacks that a company's security review asks for first. Many identity providers a
company already runs are SAML-only, and single sign-on today speaks only OpenID Connect (ADR 0186). And the audit log
is kept 45 days and can be read only through `GET /api/audit`, so a company that must keep a year of change records, or
feed them to its own log store, has nowhere to put them. Both are listed under Enterprise depth in `ROADMAP.md`. Both
are enterprise features under ADR 0184: nothing that is free today moves behind the key, and the 45-day log with
`GET /api/audit` stays as it is for every instance.

ADR 0189 (the SAML library and the one-provider rule) and ADR 0190 (the audit stream's destination and delivery) carry
the decisions, both made on 2026-11-14.

## SAML beside OIDC

### One provider, either protocol

An instance holds one single sign-on provider, and it is OIDC or SAML. Saving a SAML provider while an OIDC one is saved
is refused with a message that says to remove the OIDC one first, and the same the other way round. Why not both: the
sign-in page shows one button, SSO-only (ADR 0186) means "the provider", and two protocols would need a rule for which
one an existing account's email belongs to. Removing a provider never changes an account (ADR 0186), so changing
protocol is remove, then set up the other; people who signed in through the old provider are matched by verified email
to the new one, as a first sign-in through any provider is.

The license feature is the same `sso`: a key that unlocks one protocol unlocks the other. The permission is the same,
`instance:write` to change and `instance:read` to see.

### The flow

SP-initiated only. Nexul is the service provider; the IdP-initiated flow, where the provider posts an assertion nobody
asked for, is left out because it cannot be bound to a request Nexul made.

1. `GET /auth/saml` makes an AuthnRequest, stores its id server-side for 10 minutes and redirects the browser to the
   IdP's single sign-on URL (HTTP-Redirect binding), with a `RelayState` that is a random value.
2. The IdP posts the response to `POST /auth/saml/acs` (HTTP-POST binding). The browser's request comes from another
   site, so the state cookie that OIDC uses (`SameSite=Lax`) is not sent; the pending request is therefore found by the
   response's `InResponseTo`, in a small table, and deleted when used, which is also what stops a replayed response.
3. The response is accepted only when its assertion is signed (the assertion, or the response that wraps it, by a
   certificate in the IdP's metadata), `InResponseTo` names a pending request, the issuer is the configured entity id,
   the audience is Nexul's entity id, and the time conditions hold with at most 60 seconds of clock skew. An unsigned
   assertion is refused, never accepted because the transport was https. Encrypted assertions are not supported in the
   first version: https carries the assertion, and the encryption code is where the library's worst advisories sit.
4. The account is matched as OIDC does (ADR 0186): the sign-in identity is the pair of the IdP's entity id and the
   NameID (provider `saml`); a first sign-in whose email is the login of an existing Google or Discord sign-in links to
   that account, otherwise it makes one in the default workspace with the role the Owner chose. A transient NameID is
   refused, since it is different on every sign-in; the Owner is told to send a persistent one.
5. A session started this way carries the `sso` mark that `sessions.sso` holds (the column added by the change that ends
   a session when SSO-only turns on), so that change covers SAML sessions without a line of its own.

The email comes from the attribute the Owner names (default `email`; Entra sends a long claim URI, so any attribute
name is accepted). SAML has no `email_verified` claim. A signed assertion from the provider the Owner configured is
taken as the provider vouching for that address, which is a weaker guarantee than OIDC's explicit flag: an IdP that lets
a person type their own email attribute allows taking over a matched account. The card says this in one line next to the
attribute field, and ADR 0189 records it.

SSO-only and the Owner-passkey escape are the same: the switch hides every other method, the server refuses them, and it
turns on only while an instance Owner holds a passkey. The sign-in page reads one `Public` answer that now carries the
protocol, and the button goes to `/auth/saml` or `/auth/oidc` accordingly.

### Setting it up

The Owner enters the provider's metadata by URL (fetched by the server, https only, refreshed every 24 hours and on
demand) or by upload (the XML pasted or sent as a file, kept as it was, never refreshed). Either way the server parses
it, takes the IdP's entity id, single sign-on URL and signing certificates from it, and refuses metadata with none of
those. Metadata that arrives by URL is not itself signature-checked in the first version; the https certificate is its
only protection, and the guide says so.

Nexul publishes its own SP metadata at `GET /auth/saml/metadata`, with no session needed: the entity id (the instance
address followed by `/auth/saml/metadata`), the assertion consumer service URL (`/auth/saml/acs`) and the SP
certificate. The card shows the three values to copy and a link to the XML. The server generates the SP key pair on first
save, signs AuthnRequests with it, and never sends the private key anywhere.

Rotation has two sides. When the IdP rotates, a metadata URL picks the new certificate up on its next refresh and every
certificate in the metadata verifies a response, so both are accepted while the IdP publishes both; an upload is
replaced by uploading again. When Nexul's own certificate nears expiry, **Rotate certificate** makes a new pair and
publishes both in the SP metadata until **Retire old certificate** drops the old one; the card shows each certificate's
expiry and warns 30 days before.

The saved fields: metadata source (URL or XML), the fetched entity id, SSO URL and certificates as read-only results,
the email attribute, an optional display-name attribute, the name on the button, the role new people join as, and the
SSO-only switch. A **Verify** with named checks, as OIDC's (`POST /api/instance/saml/verify?check=`): one that the
metadata parses and names a signing certificate, one that the email attribute is non-empty. The sign-in itself is the
test of the rest.

### Every surface

- HTTP gateway: public `GET /auth/saml`, `POST /auth/saml/acs`, `GET /auth/saml/metadata` and `GET /api/auth/sso`
  (extended, not duplicated); the Owner's `GET`, `PUT`, `DELETE /api/instance/saml` and `POST
  /api/instance/saml/verify`, plus `POST /api/instance/saml/rotate` and `POST /api/instance/saml/retire`. The verify
  route joins `readOnlyRoutes` in `internal/integrations/audit.go`; every write route is audited and named in the
  permission catalog (`server/cmd/route_permissions.go`).
- MCP: `instance_update` gains a `saml` object (the metadata URL or XML, the attributes, `display_name`, `join_role_id`,
  `only`, `remove`) and `instance_get` reads it back with the SP metadata values and certificate expiries, never the
  private key; the advertised size budget is raised for the new fields, as ADR 0186 did.
- Events: `instance.saml_changed`, declared in the package's `Topics()`, payload as `instance.sso_changed` carries (display
  name, entity id, `only`, `removed`, who, when), then `make event-schemas`, written to the outbox in the same
  transaction as the settings. Certificate rotation publishes the same topic.
- Live push: an `instanceScope` rule for the topic in `liveRules`, `make live-topics`, and a follower in
  `web/src/ee/hooks/` that refreshes the card, as `instance.sso_changed` does.
- Phone app: sign-in through the browser flow the phone already uses for OIDC; no new screen.
- Search: not indexed. Permissions: the same two, no new action.
- Reverse states: remove the provider; turn SSO-only off; retire a rotated certificate; a metadata URL that stops
  answering keeps the last good certificates and the card shows the refresh failing.
- Web: the Single sign-on card in Settings gains a protocol choice at setup (OpenID Connect or SAML), the SAML form, the
  SP values with copy buttons, the certificate rows and the rotation actions; the sign-in page shows the same one button.
  Verified at 768, 1024 and 1440px.
- Guide: `website/src/content/docs/docs/guide/single-sign-on.md` gains a SAML section and loses "SAML is not part of
  it"; the website verified at 320, 375, 414 and 768px.
- `CONTEXT.md`: **Single sign-on** says OpenID Connect or SAML; **Service provider metadata** is new; **Single sign-on**'s
  `_Avoid_` line drops "SAML" and keeps "SSO provider" and "IdP login".

### The library

Parsing, signature and XML canonicalisation checking is cryptography the codebase must not write itself
(`practices/go.md`, section 15), so a library does it. Facts read from GitHub on 2026-11-14:

`github.com/crewjam/saml`: BSD-2-Clause, 1,116 stars, last push 2026-11-13, latest tag `v0.6.3` (2026-11-08), 3 open
issues, `go 1.25` module, depends on `russellhaering/goxmldsig` 1.6.1. Twelve security advisories were published on
2026-11-07, among them a critical (a crafted multi-reference signature passes verification, CVE-2026-33487), several
high (namespace-blind assertion parsing that allows NameID and attribute spoofing; service provider trusting unsigned IdP
metadata; quadratic CPU and memory on unsigned responses) and the encrypted-assertion ones. The patched versions are
0.5.2, 0.6.0 and 0.6.1, so `v0.6.3` carries all of them. Earlier advisories go back to 0.4.x. It is pre-1.0.

`github.com/russellhaering/gosaml2`: Apache-2.0, 370 stars, latest `v0.12.0` (2026-08-05), 67 open issues, last push
2026-09-02; advisories in 2020, 2022, 2023 and three on 2026-03-18 and 2026-08-19, all patched by `v0.12.0`. It is a
service-provider library only, which is all Nexul needs, and it shares `goxmldsig` (itself with a critical in 2020, a
high in March and another high in August 2026, patched in 1.6.1). `mattermost/gosaml2` is a one-star fork whose
`v0.11.0` was cut on 2026-11-13; it is not a maintained alternative. `zitadel/saml` (71 stars, `v0.4.1` of 2025-10-30)
is built around being a provider and a server, not a client.

Decided, adding the one dependency: `crewjam/saml` at `v0.6.3` or newer, used through `saml.ServiceProvider` (`MakeAuthenticationRequest`, `ParseResponse`)
and not its `samlsp` middleware, whose cookie session Nexul does not want. It is the library most Go services use, it
is under active repair, and the advisories above are a reason to pin and watch it, not to avoid it: all of them are fixed,
and the other library carries the same signature dependency. `govulncheck` runs in CI (`make vuln`), which is the watch.
The cost is a pre-1.0 API that may break on a minor bump; Dependabot's weekly grouped update is how it is followed.

## Audit log retention and streaming

### What the log holds today

One row per change: a non-GET request over the HTTP gateway, except the read-only routes listed in
`readOnlyRoutes` (`internal/integrations/audit.go`), and one row per MCP tool call whose tool is not read-only
(`AuditTool` in the same file; ADR 0138). A row is `audit_log` (`internal/platform/storage/migrations/0001_schema.sql`):
`id` (UUIDv7), `actor_type`, `actor_id`, `token_id`, `action` (`<METHOD> <path>` or `mcp <tool>`) and `created_at`. It
holds no body, no workspace, no result status and no target id beyond what the path carries. The row is appended after
the handler returns, best-effort in its own write (`Service.record`); a failed append is logged and the change stands,
so a crash between the response and the append loses that row today. Rows are read with `GET /api/audit` (`audit:read`)
through `ListAudit`, newest first, with a limit and no paging or filter. The bot-message path also appends a row
(`ChatRepo.CreateBotMessage`).

Kept 45 days: `AuditRetention` is a constant of 45 days (`internal/integrations/audit.go`, ADR 0138), a loop started at
`server/cmd/workers.go:114` (`RunAuditRetention`) deletes at start and every 24 hours, 500 rows per write, through
`DeleteBefore` over `idx_audit_created` (migration 0079). The same constant also governs the processed-event records,
published outbox rows and deploy log lines in `PurgeEvents`; the setting below changes the audit rows only.

### Retention

An enterprise setting, **Audit log kept for**, in whole days from 45 to 1,095 (the bounds are a proposal; the owner may
move them). Unset means 45, which is what every instance without the feature does today; the constant stays as the
default and the floor. `RunAuditRetention` takes a function that returns the days to keep and reads it each pass, so
`internal/integrations` never imports `ee/` (the composition root passes the function, as ADR 0186 did with the
composite MCP tool). The purge of events, outbox rows and logs keeps the 45 days.

When the key lapses, nothing is deleted that the longer setting was keeping: ADR 0184 says a lapse stops the feature and
keeps its data. The loop deletes at the saved days while it is saved, even locked, and the card says so; the Owner clears
the setting to return to 45, after which the next purge deletes what is older.

### Streaming

Each audit row is sent to one destination as it is written. Three were weighed; ADR 0190 records the choice.

An HTTPS webhook with an HMAC signature: one POST per row, JSON body, header `X-Nexul-Signature: sha256=<hex>` over the
body, computed by the same function the integrations webhook relay uses. Nexul already sends this to integrations
(`signWebhook`, ADR 0043), every log store and SIEM takes it (Splunk HEC, Datadog, Elastic, Loki through a gateway, a
company's own function), it needs no new dependency, and a receiver verifies it in four lines. Syslog, RFC 5424 over TLS:
what a traditional SIEM listens for, but a new wire format Nexul must frame (RFC 5425) and hold a connection for, with no
dependency in the tree that does it, and a person who runs a collector is as able to run an HTTP one. OTLP logs: the server
already exports its own logs this way (`logging.NewWithOTLP`, `practices/go.md`, section 3), but that sink is a slog handler
that "never blocks or fails the caller", which is the opposite of a delivery that must not lose a row, and sending audit
rows through the server's own log pipe would mix them with service logs under one set of credentials.

Decided for the first version: the HTTPS webhook with an HMAC signature (ADR 0190). The others can be added behind the
same queue later as further destination kinds.

The body of one delivery:

```json
{"delivery_id":"...","id":"<audit row id>","created_at":"2026-11-14T09:30:00Z","actor_type":"user","actor_id":"...","token_id":"","action":"PATCH /api/projects/abc","instance_url":"https://nexul.example.com"}
```

`id` is the audit row's id, the key a receiver dedupes on, since delivery is at least once and a retry arrives after
later rows. Receivers order by `created_at`. The URL must be https (http on a loopback address, as the SSO issuer rule
allows, for trying it out). The signing secret is shown once at creation and may be a secret reference, as the SSO client
secret is; **Rotate secret** makes a new one.

### Delivery

The audit write already runs through the single writer. Appending the row also inserts, in the same transaction, one
row into a new `audit_stream_deliveries` table (the audit row's id, `status` pending, `attempts`, `next_attempt_at`,
`delivered_at`, last error) when streaming is on and unlocked, which is the transactional outbox of
`practices/architecture.md`, section 3, with the audit row as the event. So a committed row is never lost, and the
request never waits on the network: the relay is a separate loop. It is the webhook relay's design
(`internal/integrations/webhook.go`): it waits on the storage commit broadcast through `store.CommitsTo` with a new
`AuditStreamWrites` entry in `internal/platform/storage/loop_writes.go` (the guard test fails until it is added), has a
5-second fallback poll, takes a batch of due rows, POSTs with a 15-second timeout, marks a row delivered only after a 2xx,
backs off exponentially between 1 and 30 seconds, and after 8 attempts marks the row `dead`. A dead row stays, shows in
the card's count with its last error, and **Retry failed** puts every dead row back to pending. Deliveries cascade
with their audit row, so retention removes both. Rows written before streaming was turned on are not sent.

A delivery that cannot reach the destination for a day does not stop anything: audit rows keep being written, deliveries
queue (bounded by the retention period), and the card shows pending and dead counts and when the last one went out.

The facts that the first version does not change: the append is still best-effort after the handler (a crash between the
response and the append loses that row, as today), and moving it into the handler's transaction is a separate change that
touches every domain, which ADR 0138 rejected.

The license feature is named `audit`, for retention and streaming together: one key covers every enterprise feature, and
the feature list is `Features` in `ee/license/model.go`. `audit` is also the permission area of `audit:read`, so the
name is the one that reads naturally and its code constant is `FeatureAudit`.

### Every surface

- Package: `ee/auditlog` (settings, the stream queue use-cases, the relay), a repo in `internal/platform/storage`, the
  queries in `queries/audit_settings.sql` and `queries/audit_stream.sql`, `make sqlc`, a migration for the two tables.
- HTTP gateway: `GET`, `PUT`, `DELETE /api/instance/audit` (settings view with counts), `POST
  /api/instance/audit/stream/test` (sends one signed sample row and reports the status; read-only, so it joins
  `readOnlyRoutes`), `POST /api/instance/audit/stream/retry`, `POST /api/instance/audit/stream/rotate-secret`; each
  names `instance:write` or `instance:read` in the permission catalog. `GET /api/audit` is unchanged but can now return
  rows older than 45 days; its description in `server/cmd/routes.go` ("kept 45 days") is updated.
- MCP: `instance_update` gains an `audit` object (`retention_days`, `stream_url`, `stream_secret`, `stream_enabled`,
  `retry_failed`, `remove`) and `instance_get` reads it back with the counts, never the secret. A read of the log itself
  stays `GET /api/audit`; no new tool, since extending `instance_update` first is the rule when the tool budget is tight (`practices/mcp.md`, section on the budget).
- Events: `instance.audit_settings_changed` (retention days, stream URL, enabled, removed, who, when; never the secret),
  in `Topics()`, `make event-schemas`, outbox in the settings transaction. The stream does not publish an event per row:
  the audit row is the record, and a topic per audit row would feed the bus with its own audit.
- Live push: an `instanceScope` rule, `make live-topics`, a follower that refreshes the card. Delivery counts refresh on
  that topic and on a window refocus, not per delivery.
- Web: a **Audit log** card in Settings, Instance, under single sign-on: days kept, the destination form with Test, the
  pending and dead counts, last delivery time, Retry failed. Verified at 768, 1024 and 1440px.
- Guide: a new page `audit-log.md` beside `enterprise-license.md`, and the audit sentence in `api-and-tokens.md`
  ("kept for 45 days") says "kept 45 days, longer with an enterprise key".
- `CONTEXT.md`: **Audit log** says kept 45 days, longer and streamed with the enterprise feature; **Audit stream** is new.
- Search, phone app, permissions beyond the two: none.
- Reverse states: clear retention, turn the stream off (queued rows stay and send when it is turned back on, up to
  retention), remove the settings, retry dead rows.

## Tickets

`.scratch/enterprise-depth/issues/` holds seven, in order: 01 to 04 are SAML, 05 to 07 are the audit log. None starts
before its ADR is decided.
