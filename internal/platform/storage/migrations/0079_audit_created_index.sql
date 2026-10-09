-- The retention purge deletes by created_at and the audit list reads newest first; both walk this index, not the table.
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at, id);
