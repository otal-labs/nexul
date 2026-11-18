-- One import of another tool's records into a project (ADR 0201). The credential is sealed with the instance key and
-- goes the moment the import ends, which the CHECK holds every final state to.
CREATE TABLE IF NOT EXISTS imports (
    id                   TEXT PRIMARY KEY,
    workspace_id         TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id           TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source               TEXT NOT NULL,
    source_from          TEXT NOT NULL DEFAULT '',
    state                TEXT NOT NULL CHECK (state IN ('scanning', 'ready', 'importing', 'completed', 'failed', 'cancelled')),
    created_by           TEXT NOT NULL,
    credential_encrypted TEXT,
    total                INTEGER NOT NULL DEFAULT 0,
    done                 INTEGER NOT NULL DEFAULT 0,
    failed               INTEGER NOT NULL DEFAULT 0,
    report_json          TEXT NOT NULL DEFAULT '{}',
    created_at           INTEGER NOT NULL,
    updated_at           INTEGER NOT NULL,
    CHECK (state NOT IN ('completed', 'failed', 'cancelled') OR credential_encrypted IS NULL)
);
-- One open import per workspace at a time; serves the refusal of a second.
CREATE UNIQUE INDEX IF NOT EXISTS idx_imports_open_workspace ON imports(workspace_id) WHERE state IN ('scanning', 'ready', 'importing');
-- Serves the import loop's pass over the open imports, oldest first.
CREATE INDEX IF NOT EXISTS idx_imports_open ON imports(state, created_at, id) WHERE state IN ('scanning', 'ready', 'importing');
-- Serves a workspace's imports, newest first.
CREATE INDEX IF NOT EXISTS idx_imports_workspace ON imports(workspace_id, created_at, id);

-- What one import has done with each record it read, so a restart resumes where it stopped.
CREATE TABLE IF NOT EXISTS import_items (
    import_id TEXT NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
    kind      TEXT NOT NULL,
    source_id TEXT NOT NULL,
    nexul_id  TEXT NOT NULL DEFAULT '',
    state     TEXT NOT NULL CHECK (state IN ('pending', 'created', 'filled', 'failed')),
    error     TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (import_id, kind, source_id)
);

-- Every record any import brought into a workspace, so a later import of the same source skips it.
CREATE TABLE IF NOT EXISTS import_source_ids (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    source       TEXT NOT NULL,
    kind         TEXT NOT NULL,
    source_id    TEXT NOT NULL,
    nexul_id     TEXT NOT NULL,
    UNIQUE (workspace_id, source, kind, source_id)
);

-- An imported ticket's reporter names the tool it came from.
ALTER TABLE tickets ADD COLUMN reporter_source TEXT NOT NULL DEFAULT '';
