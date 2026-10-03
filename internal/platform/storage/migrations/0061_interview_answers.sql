-- A project's interview answers, one per question per round, kept apart from the interview memory so deleting it keeps them.
CREATE TABLE IF NOT EXISTS interview_answers (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    round        INTEGER NOT NULL CHECK (round >= 0),
    question     TEXT NOT NULL,
    options      TEXT,
    multi_select INTEGER NOT NULL DEFAULT 0,
    why          TEXT,
    selected     TEXT NOT NULL DEFAULT '[]',
    free_text    TEXT NOT NULL DEFAULT '',
    skipped      INTEGER NOT NULL DEFAULT 0,
    answered_by  TEXT NOT NULL DEFAULT '',
    answered_at  INTEGER NOT NULL
);
-- Serves the save upsert, the clear, and ListInterviewAnswers' project lookup.
CREATE UNIQUE INDEX IF NOT EXISTS idx_interview_answers_question ON interview_answers(project_id, round, question);
