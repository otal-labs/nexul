-- One person's emoji on one message; the key makes reacting twice a no-op and serves ListMessageReactions' message_id IN lookup.
CREATE TABLE IF NOT EXISTS message_reactions (
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    emoji      TEXT NOT NULL,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (message_id, emoji, user_id)
);
