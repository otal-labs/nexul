# The audit log streams to one signed HTTPS webhook, from a delivery queue written with the row

The audit log is kept 45 days and read only through `GET /api/audit` (ADR 0138). The enterprise edition keeps it longer
and sends each row out as it is written, behind a license feature named `audit`. Nothing free moves behind it: the 45
days and the endpoint stay for every instance (ADR 0184).

- **Retention.** An Owner setting in whole days from 45 to 1,095, with the 45-day constant as the default and the floor. The
  purge reads the days each pass; the processed-event, outbox and deploy-log purges keep 45. A lapsed key keeps the
  saved days, because a lapse stops the feature and keeps its data (ADR 0184); the Owner clears the setting to return to 45.
- **The destination is an HTTPS webhook.** One POST per audit row, JSON body, `X-Nexul-Signature: sha256=<hex>` over the
  body, from the function the integrations webhook already uses (ADR 0043). The body carries the row's `id`, which a
  receiver dedupes on, since delivery is at least once. Rejected: syslog over TLS (RFC 5424, framed by RFC 5425), which
  needs a new wire format and a held connection with no dependency in the tree for either, for a reader a company can
  put an HTTP front on as easily; and OTLP logs, which the server already exports (`practices/go.md`, section 3), but
  through a slog handler that by design never blocks or fails the caller, the opposite of delivery that must not lose
  a row, and under the same credentials as the service's own logs. Other kinds can sit behind the same queue later.
- **Delivery is the transactional outbox with the audit row as the event.** `AuditRepo.Append` also inserts a row in
  `audit_stream_deliveries` in the same transaction when the stream is on and unlocked, so a committed row is never lost
  and the request never waits on the network. A relay shaped like the webhook relay delivers it: it wakes on the commit
  broadcast through `AuditStreamWrites` in `loop_writes.go`, polls every 5 seconds as the retry path, marks a row
  delivered only on a 2xx, backs off from 1 to 30 seconds, and marks it dead after 8 attempts. A dead row stays with its
  error and returns to pending on an Owner's retry. Deliveries cascade with their audit row, so retention removes both.
  Rows written before the stream was turned on are not sent.
- **What it does not fix.** The append itself is still best-effort after the handler returns, so a crash between the
  response and the append loses that row, as before. Moving it into each handler's transaction touches every domain,
  which ADR 0138 rejected, and this change leaves that open.
- **Dependencies.** None added: the HTTP client, HMAC and queue are in the tree.

Decided: pending owner approval.
