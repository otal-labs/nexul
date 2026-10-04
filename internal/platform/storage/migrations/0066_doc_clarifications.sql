-- A doc's clarification: one row per round a Clarify run opened. Closed lives on the newest round, so a new round reopens it.
CREATE TABLE IF NOT EXISTS doc_clarification_rounds (
    doc_id              TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    round               INTEGER NOT NULL CHECK (round >= 1),
    started_by          TEXT NOT NULL,
    trail_id            TEXT NOT NULL DEFAULT '',
    started_at          INTEGER NOT NULL,
    running             INTEGER NOT NULL DEFAULT 1,
    took_lock           INTEGER NOT NULL DEFAULT 0,
    anything_else       TEXT NOT NULL DEFAULT '',
    anything_else_by    TEXT NOT NULL DEFAULT '',
    anything_else_at    INTEGER NOT NULL DEFAULT 0,
    anything_else_reply TEXT NOT NULL DEFAULT '',
    no_gaps_at          INTEGER NOT NULL DEFAULT 0,
    closed_by           TEXT NOT NULL DEFAULT '',
    closed_at           INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (doc_id, round)
);

-- One question a round asked; no picks, no text and not skipped is pending.
CREATE TABLE IF NOT EXISTS doc_clarification_questions (
    id           TEXT PRIMARY KEY,
    doc_id       TEXT NOT NULL,
    round        INTEGER NOT NULL,
    position     INTEGER NOT NULL,
    question     TEXT NOT NULL,
    why          TEXT NOT NULL DEFAULT '',
    options      TEXT NOT NULL DEFAULT '[]',
    multi_select INTEGER NOT NULL DEFAULT 0,
    selected     TEXT NOT NULL DEFAULT '[]',
    free_text    TEXT NOT NULL DEFAULT '',
    skipped      INTEGER NOT NULL DEFAULT 0,
    answered_by  TEXT NOT NULL DEFAULT '',
    answered_at  INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (doc_id, round) REFERENCES doc_clarification_rounds(doc_id, round) ON DELETE CASCADE
);
-- Serves ListDocClarificationQuestions, CountPendingDocClarificationQuestions, the round cascade, and keeps a round from asking one question twice.
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_clarification_questions_question ON doc_clarification_questions(doc_id, round, question);
