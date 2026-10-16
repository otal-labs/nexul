-- A service's stretch of being down: opened by a container dying or being OOM-killed, closed once it has run steadily.
CREATE TABLE IF NOT EXISTS service_outages (
    id            TEXT PRIMARY KEY,
    stack_id      TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    service       TEXT NOT NULL,
    container     TEXT NOT NULL,
    started_at    INTEGER NOT NULL,
    exit_code     INTEGER NOT NULL DEFAULT 0,
    oom           INTEGER NOT NULL DEFAULT 0,
    restart_count INTEGER NOT NULL DEFAULT 0,
    ended_at      INTEGER
);
-- Serves GetOpenOutage and ListOpenOutagesByStack, and refuses a second open outage for one service.
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_outages_open ON service_outages(stack_id, service) WHERE ended_at IS NULL;
-- Serves LatestOutageEnd, the newest closed outage of one service.
CREATE INDEX IF NOT EXISTS idx_service_outages_ended ON service_outages(stack_id, service, ended_at);
