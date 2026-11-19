-- A doc folder may sit inside another folder of its project; '' is a top-level folder, and every existing folder is one.
ALTER TABLE doc_folders ADD COLUMN parent_id TEXT NOT NULL DEFAULT '';

-- A folder name is unique among its siblings ignoring case, so two folders may share a name under different parents.
DROP INDEX IF EXISTS idx_doc_folders_name;
-- Serves CreateDocFolder and UpdateDocFolder's duplicate check and the top-level walk of ListDocRuleTurns.
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_folders_name ON doc_folders(project_id, parent_id, name COLLATE NOCASE);
-- Serves the walks down a folder's subtree: DeleteDocFolder's, ListDocFolderSubtree's and ListDocRuleTurns'.
CREATE INDEX IF NOT EXISTS idx_doc_folders_parent ON doc_folders(parent_id) WHERE parent_id != '';
