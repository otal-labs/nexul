-- A project's docs live in folders one level deep, each project with one default folder that is never deleted (ADR 0096).
CREATE TABLE IF NOT EXISTS doc_folders (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    is_default  INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
-- Serves CreateDocFolder and RenameDocFolder's duplicate check, and ListDocFoldersByProject by its project prefix: a name is unique per project ignoring case, as typed.
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_folders_name ON doc_folders(project_id, name COLLATE NOCASE);
-- Serves GetDefaultDocFolder, and keeps a project to one default folder.
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_folders_default ON doc_folders(project_id) WHERE is_default = 1;

-- Every project from before folders gets its default folder, Main, which holds all of its existing docs.
INSERT INTO doc_folders (id, project_id, name, is_default, created_at, updated_at)
SELECT lower(hex(randomblob(16))), p.id, 'Main', 1, strftime('%s','now'), strftime('%s','now')
FROM projects p
WHERE NOT EXISTS (SELECT 1 FROM doc_folders f WHERE f.project_id = p.id AND f.is_default = 1);

ALTER TABLE docs ADD COLUMN folder_id TEXT NOT NULL DEFAULT '';
UPDATE docs SET folder_id = COALESCE((SELECT f.id FROM doc_folders f WHERE f.project_id = docs.project_id AND f.is_default = 1), '')
WHERE folder_id = '';
-- Serves DeleteDocFolder's move of a folder's docs into the default folder.
CREATE INDEX IF NOT EXISTS idx_docs_folder ON docs(folder_id);
