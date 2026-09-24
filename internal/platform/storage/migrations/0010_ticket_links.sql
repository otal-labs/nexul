CREATE TABLE IF NOT EXISTS ticket_links (
    ticket_id   TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    target_id   TEXT REFERENCES tickets(id) ON DELETE CASCADE,
    created_at  INTEGER NOT NULL
);
-- One found-in per ticket, a known origin or the NULL-target origin-unknown marker; serves DeleteTicketFoundIn.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_links_found_in ON ticket_links(ticket_id) WHERE kind = 'found_in';
-- Serves ListTicketLinksFrom, ListTicketBlockerIDs, and DeleteTicketBlocker; also refuses a duplicate blocker.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_links_from ON ticket_links(ticket_id, kind, target_id);
-- Serves ListTicketLinksTo (blocks, bugs found in this) and ListUnclearedTicketBlockers.
CREATE INDEX IF NOT EXISTS idx_ticket_links_to ON ticket_links(target_id, kind);
