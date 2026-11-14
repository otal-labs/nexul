# The audit stream: the queue and the relay

**Status:** ready-for-agent
**Blocked by:** 05

Each audit row is signed and posted to the destination as it is written, and is never lost or waited on.

## Done looks like

- A migration for `audit_stream_deliveries` (audit id with `ON DELETE CASCADE`, status, attempts, `next_attempt_at`,
  `delivered_at`, last error) and a due index. `AuditRepo.Append` inserts the delivery row in the same transaction when the
  stream is enabled and unlocked, and not otherwise.
- A relay in `ee/auditlog` modelled on the webhook relay: waits on `store.CommitsTo(AuditStreamWrites)` with a 5-second
  fallback, batches 50 due rows, POSTs the JSON body with `X-Nexul-Signature` from the shared signing function, 15-second
  timeout, 2xx marks delivered, otherwise backs off between 1 and 30 seconds, dead after 8 attempts. Started from
  `server/cmd/workers.go` with a shutdown path.
- `AuditStreamWrites` added to `internal/platform/storage/loop_writes.go`.
- `POST /api/instance/audit/stream/test` sends one signed sample and returns the receiver's status without queueing;
  `POST .../retry` returns dead rows to pending; `POST .../rotate-secret` makes a new one. The view carries pending and
  dead counts, the last delivery time and the last error.

## Tests

Error paths first: a destination that returns 500, 4xx, a redirect, never answers (timeout), refuses the connection; a
crash between commit and delivery (a new relay delivers the pending row); a row delivered twice is acceptable and
carries the same `id`; a row going dead at 8 attempts and returning on retry; the destination removed while rows are
queued; a locked key stops queueing and keeps queued rows. Then: the signature verifies against the body, delivery
order within a batch follows `created_at`, a burst of audit writes never makes a request wait (an `Append` test with a
hanging destination finishes), `TestLoopWrites_CoverEveryTableTheLoopsPassReads` passes, no goroutine outlives the
package's tests. Time-based loops use `testing/synctest`.
