-- The audit stream's queue (ADR 0190): one row per audit row written while the stream was on and unlocked, inserted in
-- the same transaction as the audit row and removed with it by retention. status is pending, delivered or dead.
CREATE TABLE IF NOT EXISTS audit_stream_deliveries (
    audit_id        TEXT    PRIMARY KEY REFERENCES audit_log (id) ON DELETE CASCADE,
    status          TEXT    NOT NULL DEFAULT 'pending',
    attempts        INTEGER NOT NULL DEFAULT 0,
    next_attempt_at INTEGER NOT NULL,
    delivered_at    INTEGER,
    last_error      TEXT    NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL
);

-- AuditStreamDue reads the pending rows whose time has come; CountAuditStream and LastAuditStreamError count and read
-- the pending and dead ones without walking the delivered.
CREATE INDEX IF NOT EXISTS idx_audit_stream_status ON audit_stream_deliveries (status, next_attempt_at);
-- LastAuditStreamDelivery: the newest delivery time, read from the end of the index.
CREATE INDEX IF NOT EXISTS idx_audit_stream_delivered ON audit_stream_deliveries (delivered_at) WHERE delivered_at IS NOT NULL;
