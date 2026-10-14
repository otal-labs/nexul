-- A workspace is archived rather than deleted: NULL while live, the archive time once archived, cleared by a restore.
ALTER TABLE workspaces ADD COLUMN archived_at INTEGER;
