-- A stack's error-rate rule: off for every stack that exists, with the numbers a team starts from.
ALTER TABLE stacks ADD COLUMN error_rate_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE stacks ADD COLUMN error_rate_percent REAL NOT NULL DEFAULT 5;
ALTER TABLE stacks ADD COLUMN error_rate_minutes INTEGER NOT NULL DEFAULT 5;
ALTER TABLE stacks ADD COLUMN error_rate_min_requests INTEGER NOT NULL DEFAULT 20;

-- A stretch of a stack answering too many requests with 5xx: opened when a window crosses its rule, closed once the
-- share has stayed under the limit. Peak, requests and errors are those of the worst window seen; the rule's numbers
-- are the ones in force when it opened.
CREATE TABLE IF NOT EXISTS error_rate_episodes (
    id             TEXT PRIMARY KEY,
    stack_id       TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    started_at     INTEGER NOT NULL,
    peak_percent   REAL NOT NULL,
    requests       INTEGER NOT NULL,
    errors         INTEGER NOT NULL,
    limit_percent  REAL NOT NULL,
    window_minutes INTEGER NOT NULL,
    min_requests   INTEGER NOT NULL,
    ended_at       INTEGER
);
-- Serves GetOpenErrorRateEpisode, and refuses a second open episode for one stack.
CREATE UNIQUE INDEX IF NOT EXISTS idx_error_rate_episodes_open ON error_rate_episodes(stack_id) WHERE ended_at IS NULL;
-- Serves LatestErrorRateEnd, the newest closed episode of one stack.
CREATE INDEX IF NOT EXISTS idx_error_rate_episodes_ended ON error_rate_episodes(stack_id, ended_at);
