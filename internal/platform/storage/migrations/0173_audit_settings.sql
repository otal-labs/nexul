-- The audit log's enterprise settings, one row: the days rows are kept (0 is unset, the free 45) and the one stream
-- destination with its signing secret, sealed, or a secret reference as typed (ADR 0190). Kept when the license key
-- lapses, so the saved days still hold and a renewed key turns the stream back on.
CREATE TABLE IF NOT EXISTS audit_settings (
    id             INTEGER PRIMARY KEY CHECK (id = 1),
    retention_days INTEGER NOT NULL DEFAULT 0,
    stream_url     TEXT    NOT NULL DEFAULT '',
    stream_secret  TEXT    NOT NULL DEFAULT '',
    stream_enabled INTEGER NOT NULL DEFAULT 0,
    updated_by     TEXT    NOT NULL DEFAULT '',
    updated_at     INTEGER NOT NULL
);
