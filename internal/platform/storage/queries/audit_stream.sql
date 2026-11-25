-- name: InsertAuditStreamDelivery :exec
INSERT INTO audit_stream_deliveries (audit_id, next_attempt_at, created_at) VALUES (?, ?, ?);

-- name: AuditStreamDue :many
-- The pending rows due by now, oldest audit row first, which is the order a receiver reads them in.
SELECT d.audit_id, d.attempts, a.actor_type, a.actor_id, a.token_id, a.action, a.created_at
FROM audit_stream_deliveries d JOIN audit_log a ON a.id = d.audit_id
WHERE d.status = 'pending' AND d.next_attempt_at <= sqlc.arg(now)
ORDER BY a.created_at, a.id LIMIT sqlc.arg(max_rows);

-- name: MarkAuditStreamDelivered :exec
UPDATE audit_stream_deliveries SET status = 'delivered', delivered_at = ?, last_error = '' WHERE audit_id = ?;

-- name: MarkAuditStreamFailed :exec
UPDATE audit_stream_deliveries SET attempts = ?, next_attempt_at = ?, last_error = ? WHERE audit_id = ?;

-- name: MarkAuditStreamDead :exec
UPDATE audit_stream_deliveries SET status = 'dead', attempts = ?, last_error = ? WHERE audit_id = ?;

-- name: RetryAuditStreamDead :execrows
UPDATE audit_stream_deliveries SET status = 'pending', attempts = 0, next_attempt_at = ? WHERE status = 'dead';

-- name: CountAuditStream :many
SELECT status, COUNT(*) AS n FROM audit_stream_deliveries WHERE status IN ('pending', 'dead') GROUP BY status;

-- name: LastAuditStreamDelivery :one
SELECT MAX(delivered_at) FROM audit_stream_deliveries;

-- name: LastAuditStreamError :one
-- The error of the row that failed last: a dead one keeps its last retry time, a pending one its next.
SELECT last_error FROM audit_stream_deliveries
WHERE status IN ('pending', 'dead') AND last_error != ''
ORDER BY next_attempt_at DESC LIMIT 1;
