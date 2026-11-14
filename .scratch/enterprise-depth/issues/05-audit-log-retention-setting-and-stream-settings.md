# Audit log retention setting and the stream's settings

**Status:** ready-for-agent
**Blocked by:** ADR 0190 decided (the destination and delivery)

The Owner can keep the audit log longer and save one stream destination. Nothing is sent yet.

## Done looks like

- `FeatureAudit` ("audit") in `ee/license/model.go`'s `Features`, so a key can unlock it.
- A migration for `audit_settings` (retention days, stream URL, secret, enabled, updated by and at). `ee/auditlog` holds
  the use-cases; the repo lives in `internal/platform/storage` with sqlc queries.
- `RunAuditRetention` takes a function returning the days to keep, read each pass, defaulting to the 45-day constant, with
  the constant as the floor. `PurgeEvents` keeps 45. A lapsed key does not shorten the saved days.
- `GET`, `PUT`, `DELETE /api/instance/audit`; `instance_update` takes an `audit` object and `instance_get` reads it back
  without the secret. Days outside 45 to 1,095 and a non-https stream URL (loopback http excepted) are refused.
- `instance.audit_settings_changed` declared, in the outbox with the write, schema generated, live rule and follower stub.
- `GET /api/audit`'s description and `CONTEXT.md`'s Audit log term stop saying 45 days flat.

## Tests

Error paths first: days below 45 and above the cap, zero, negative; a URL that is http, has credentials or no host; a
secret reference that does not resolve; writing with the key locked; a read with the key locked still works. Then: the
purge deletes at the saved days and at 45 when unset, in 500-row batches; a lapsed key keeps the saved days; clearing the
setting returns to 45 and the next purge deletes the older rows; the events purge is unchanged by the setting. The purge
test extends `audit_test.go` with the same fake clock. Parity: permission catalog, audit classification (the PUT is
audited, a test delivery is not), event contract.
