-- The receipt of each owner's one notice to move their tunnel or URL computers to the Nexul app (ADR 0146): its row
-- keeps the notice to once per owner across restarts, and notified_at starts that owner's 30 days.
CREATE TABLE IF NOT EXISTS pairing_move_notices (
    user_id     TEXT PRIMARY KEY,
    notified_at INTEGER NOT NULL
);
