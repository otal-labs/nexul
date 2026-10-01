-- Memories are project-scoped only (ADR 0099): workspace-scoped ones are deleted, with their attachments and
-- versions removed explicitly rather than left to the foreign keys' cascades. The use-case refuses a memory with no
-- project, so the column stays nullable and the table is not rebuilt.
DELETE FROM attachments WHERE memory_id IN (SELECT id FROM memories WHERE project_id IS NULL);
DELETE FROM memory_versions WHERE memory_id IN (SELECT id FROM memories WHERE project_id IS NULL);
DELETE FROM memories WHERE project_id IS NULL;
DROP INDEX IF EXISTS idx_memories_workspace_scoped;
