-- Everyone who can read a stack watches it; a row here is someone who turned that off.
CREATE TABLE IF NOT EXISTS stack_unwatchers (
    stack_id   TEXT NOT NULL REFERENCES stacks(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (stack_id, user_id)
);
