-- A project's interview sources; refs carry no foreign key, so a deleted doc, memory, or project reads as gone.
CREATE TABLE IF NOT EXISTS interview_sources (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('path', 'doc', 'memory', 'project', 'text')),
    ref          TEXT NOT NULL DEFAULT '',
    label        TEXT NOT NULL DEFAULT '',
    body         TEXT NOT NULL DEFAULT '',
    stance       TEXT NOT NULL CHECK (stance IN ('follow', 'question')),
    added_by     TEXT NOT NULL,
    added_at     INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);
-- Serves ListInterviewSources' ordered read and CountInterviewSources.
CREATE INDEX IF NOT EXISTS idx_interview_sources_project ON interview_sources(project_id, added_at, id);
-- Keeps a project from pointing at the same thing twice; pasted text may repeat.
CREATE UNIQUE INDEX IF NOT EXISTS idx_interview_sources_ref ON interview_sources(project_id, kind, ref) WHERE kind <> 'text';

-- A drafting run's proposed answer to a template question, kept apart from interview_answers, where a row means answered.
CREATE TABLE IF NOT EXISTS interview_drafts (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    question     TEXT NOT NULL,
    selected     TEXT NOT NULL DEFAULT '[]',
    free_text    TEXT NOT NULL DEFAULT '',
    source_ids   TEXT NOT NULL DEFAULT '[]',
    where_line   TEXT NOT NULL DEFAULT '',
    trail_id     TEXT NOT NULL DEFAULT '',
    drafted_by   TEXT NOT NULL DEFAULT '',
    drafted_at   INTEGER NOT NULL
);
-- Serves the save upsert and ListInterviewDrafts, one draft per question.
CREATE UNIQUE INDEX IF NOT EXISTS idx_interview_drafts_question ON interview_drafts(project_id, question);
