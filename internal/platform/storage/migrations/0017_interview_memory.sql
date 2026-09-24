ALTER TABLE memories ADD COLUMN kind TEXT NOT NULL DEFAULT '';
-- One memory of each special kind per project (ADR 0065); serves GetMemoryByProjectKind.
CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_project_kind ON memories(project_id, kind) WHERE kind <> '';
CREATE TABLE IF NOT EXISTS interview_templates (
    workspace_id TEXT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    body         TEXT NOT NULL,
    updated_by   TEXT NOT NULL DEFAULT '',
    updated_at   INTEGER NOT NULL
);
