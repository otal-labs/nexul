-- The workspace calendar (ADR 0168): meetings, who is invited and their replies, ticket due dates, and the reminders
-- already sent for them.

-- A repeating meeting is one row and its occurrences are computed in its timezone. A row with a series_id is one
-- occurrence of that series changed on its own, standing in for the occurrence starting at replaces_at.
CREATE TABLE IF NOT EXISTS meetings (
    id               TEXT PRIMARY KEY,
    workspace_id     TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    organiser_id     TEXT NOT NULL,
    title            TEXT NOT NULL,
    notes            TEXT NOT NULL DEFAULT '',
    starts_at        INTEGER NOT NULL,
    ends_at          INTEGER NOT NULL,
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    voice_channel_id TEXT NOT NULL DEFAULT '',
    link             TEXT NOT NULL DEFAULT '',
    repeats          TEXT NOT NULL DEFAULT 'none',
    repeat_until     INTEGER,
    remind_minutes   INTEGER NOT NULL DEFAULT 10,
    series_id        TEXT NOT NULL DEFAULT '',
    replaces_at      INTEGER,
    cancelled_at     INTEGER,
    next_remind_at   INTEGER,
    next_occurrence  INTEGER,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
);

-- A workspace's meetings that may fall in a range: single ones by start, a series until it ends.
CREATE INDEX IF NOT EXISTS idx_meetings_workspace_start ON meetings(workspace_id, starts_at, id);
-- The occurrences a series' own rows replace, read whenever the series is expanded.
CREATE INDEX IF NOT EXISTS idx_meetings_series ON meetings(series_id, replaces_at) WHERE series_id != '';
-- The reminder loop's next wake and the rows due at it.
CREATE INDEX IF NOT EXISTS idx_meetings_next_remind ON meetings(next_remind_at) WHERE next_remind_at IS NOT NULL;

-- The people invited to a meeting besides its organiser; reply is pending, accepted or declined.
CREATE TABLE IF NOT EXISTS meeting_invitees (
    meeting_id TEXT NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    reply      TEXT NOT NULL DEFAULT 'pending',
    replied_at INTEGER,
    PRIMARY KEY (meeting_id, user_id)
);
-- The meetings a person is invited to, for their calendar.
CREATE INDEX IF NOT EXISTS idx_meeting_invitees_user ON meeting_invitees(user_id, meeting_id);

-- A person's IANA timezone as their app last reported it, for the reminders that fall on a local day.
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT '';

-- A ticket's due date as YYYY-MM-DD, empty when it has none.
ALTER TABLE tickets ADD COLUMN due_on TEXT NOT NULL DEFAULT '';
-- Tickets due in a range: the calendar's due tickets and the due reminder loop.
CREATE INDEX IF NOT EXISTS idx_tickets_due_on ON tickets(due_on, id) WHERE due_on != '';

-- The due date each ticket's developer was last reminded of, so a reminder goes once per date.
CREATE TABLE IF NOT EXISTS ticket_due_reminders (
    ticket_id TEXT PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
    due_on    TEXT NOT NULL,
    sent_at   INTEGER NOT NULL
);
