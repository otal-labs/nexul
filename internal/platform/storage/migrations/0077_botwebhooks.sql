-- A bot posts into one conversation through its webhook URL and goes with it; its token is sealed with the instance key.
CREATE TABLE IF NOT EXISTS botwebhooks (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    avatar          TEXT NOT NULL DEFAULT '',
    token           TEXT NOT NULL,
    created_by      TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    last_post_at    INTEGER,
    post_count      INTEGER NOT NULL DEFAULT 0,
    deleted_at      INTEGER,
    deleted_by      TEXT NOT NULL DEFAULT ''
);
-- Serves ListBotwebhooks: a conversation's bots, oldest first.
CREATE INDEX IF NOT EXISTS idx_botwebhooks_conversation ON botwebhooks(conversation_id, created_at, id);
-- A live bot's name is its conversation's alone; a deleted one frees it.
CREATE UNIQUE INDEX IF NOT EXISTS idx_botwebhooks_live_name ON botwebhooks(conversation_id, name COLLATE NOCASE) WHERE deleted_at IS NULL;

-- A bot is a message's author without being a user (ADR 0129): the author foreign key goes, and a bot message keeps
-- the name and avatar it showed, and its embeds, as JSON.
CREATE TABLE messages_new (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    author_id       TEXT NOT NULL,
    body            TEXT NOT NULL,
    mentions        TEXT NOT NULL DEFAULT '[]',
    attachment_id   TEXT,
    edited_at       INTEGER,
    deleted_at      INTEGER,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    author_kind     TEXT NOT NULL DEFAULT 'user',
    handoffs        TEXT,
    via             TEXT NOT NULL DEFAULT '',
    author_name     TEXT,
    author_avatar_url TEXT,
    embeds          TEXT
);
INSERT INTO messages_new (id, conversation_id, author_id, body, mentions, attachment_id, edited_at, deleted_at, created_at, updated_at, author_kind, handoffs, via)
SELECT id, conversation_id, author_id, body, mentions, attachment_id, edited_at, deleted_at, created_at, updated_at, author_kind, handoffs, via FROM messages;
DROP TABLE messages;
ALTER TABLE messages_new RENAME TO messages;
-- Serves ListMessages and ListMessagesSince: a conversation's messages by time.
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at);
-- Dropped with the old table; CreateNote inserts the file before the message, so the file is there when the message links it.
CREATE TRIGGER IF NOT EXISTS ticket_notes_fts_message_ai AFTER INSERT ON messages WHEN new.attachment_id IS NOT NULL AND new.deleted_at IS NULL BEGIN
    INSERT INTO ticket_notes_fts(ticket_id, attachment_id, body)
    SELECT c.ticket_id, a.id, CAST(a.data AS TEXT) FROM attachments a JOIN conversations c ON c.id = new.conversation_id
    WHERE a.id = new.attachment_id AND c.ticket_id IS NOT NULL;
END;
