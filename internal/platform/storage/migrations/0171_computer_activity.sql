-- An action on a person's computer names it, its owner and any command it started; only Read computer activity reads
-- such a row, and the audit log leaves it out.
ALTER TABLE audit_log ADD COLUMN computer_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN computer_owner_id TEXT NOT NULL DEFAULT '';
ALTER TABLE audit_log ADD COLUMN job_id TEXT NOT NULL DEFAULT '';

-- The computer activity view, newest first (ListComputerActivity).
CREATE INDEX IF NOT EXISTS idx_audit_log_computer ON audit_log(created_at DESC, id DESC) WHERE computer_id != '';
