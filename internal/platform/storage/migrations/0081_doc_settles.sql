-- One row per doc whose person-made edits have not yet settled: doc.settled fires once due_at passes with no newer
-- save. first marks a doc created inside the window, so its settle reports a create rather than a change.
CREATE TABLE IF NOT EXISTS doc_settles (
    doc_id   TEXT PRIMARY KEY REFERENCES docs(id) ON DELETE CASCADE,
    due_at   INTEGER NOT NULL,
    actor_id TEXT NOT NULL,
    first    INTEGER NOT NULL DEFAULT 0
);
-- Serves NextDocSettleDue, ListDueDocSettles and DeleteDueDocSettles.
CREATE INDEX IF NOT EXISTS idx_doc_settles_due ON doc_settles(due_at);
