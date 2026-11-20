-- A message its author wrote to post in a conversation later, once or on a repeat, seen only by the author until it posts.
CREATE TABLE IF NOT EXISTS scheduled_messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    author_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body            TEXT NOT NULL,
    starts_at       INTEGER NOT NULL,
    send_at         INTEGER NOT NULL,
    repeats         TEXT NOT NULL DEFAULT 'none',
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);
-- Serves ScheduledMessagesDue and ScheduledMessageNext, the send loop's reads.
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_send ON scheduled_messages(send_at, id);
-- Serves ScheduledMessagesPage and ScheduledMessagesCount: an author's scheduled messages in one conversation.
CREATE INDEX IF NOT EXISTS idx_scheduled_messages_author ON scheduled_messages(conversation_id, author_id, send_at, id);
