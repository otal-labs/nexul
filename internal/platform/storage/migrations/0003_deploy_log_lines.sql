CREATE TABLE deploy_log_lines (
    seq       INTEGER PRIMARY KEY AUTOINCREMENT,
    deploy_id TEXT NOT NULL REFERENCES deploys(id) ON DELETE CASCADE,
    ts        INTEGER NOT NULL,
    phase     TEXT NOT NULL DEFAULT '',
    line      TEXT NOT NULL
);
CREATE INDEX idx_deploy_log_lines_deploy ON deploy_log_lines(deploy_id, ts, seq);

ALTER TABLE deploys DROP COLUMN log;
