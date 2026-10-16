-- A stack's requests per minute, kept a day. A minute nobody could count is a row with NULL counts, never zero.
CREATE TABLE IF NOT EXISTS stack_request_minutes (
    stack_id   TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    minute     INTEGER NOT NULL,
    requests   INTEGER,
    errors_5xx INTEGER,
    PRIMARY KEY (stack_id, minute)
) WITHOUT ROWID;
-- Serves PruneStackRequestMinutes, which drops every stack's minutes older than a day.
CREATE INDEX IF NOT EXISTS idx_stack_request_minutes_minute ON stack_request_minutes(minute);
