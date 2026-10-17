-- A stack's request minutes are kept per source, so a Cloudflare re-read replaces only its own count and never the
-- proxy's. Existing rows go to the tunnel source for a stack with a tunnel hostname, else to the proxy.
CREATE TABLE stack_request_minutes_new (
    stack_id   TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    source     TEXT NOT NULL,
    minute     INTEGER NOT NULL,
    requests   INTEGER,
    errors_5xx INTEGER,
    PRIMARY KEY (stack_id, source, minute)
) WITHOUT ROWID;
INSERT INTO stack_request_minutes_new (stack_id, source, minute, requests, errors_5xx)
SELECT m.stack_id,
       CASE WHEN EXISTS (
           SELECT 1 FROM dns_exposures e
           JOIN dns_gateways g ON g.id = e.gateway_id AND g.kind = 'tunnel'
           JOIN services s ON s.id = e.service_id
           WHERE s.stack_id = m.stack_id
       ) THEN 'tunnel' ELSE 'proxy' END,
       m.minute, m.requests, m.errors_5xx
FROM stack_request_minutes m;
DROP TABLE stack_request_minutes;
ALTER TABLE stack_request_minutes_new RENAME TO stack_request_minutes;
-- Serves PruneStackRequestMinutes, which drops every stack's minutes older than a day.
CREATE INDEX idx_stack_request_minutes_minute ON stack_request_minutes(minute);
