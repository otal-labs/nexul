-- A computer shared with a person under Run agents that they put in their own run order, in the same position sequence
-- as their own computers (ADR 0211). Ending the grant ends the row.
CREATE TABLE IF NOT EXISTS pairing_order_shares (
    user_id     TEXT NOT NULL,
    computer_id TEXT NOT NULL REFERENCES pairing_computers(id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    created_at  INTEGER NOT NULL,
    PRIMARY KEY (user_id, computer_id)
);
