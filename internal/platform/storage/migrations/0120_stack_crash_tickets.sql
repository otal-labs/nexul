-- A stack's switch to open a ticket in its project when a service crashes; off for every stack that exists.
ALTER TABLE stacks ADD COLUMN crash_tickets INTEGER NOT NULL DEFAULT 0;
-- The ticket a stack's subject (a crashed service's name) has open. The primary key serves GetStackCrashTicket and the
-- upsert; a deleted stack or ticket takes its row with it, so the next crash files a fresh ticket.
CREATE TABLE IF NOT EXISTS stack_crash_tickets (
    stack_id  TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    subject   TEXT NOT NULL,
    ticket_id TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    PRIMARY KEY (stack_id, subject)
);
