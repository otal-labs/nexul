-- The ticket lists read in created_at order; walking these indexes replaces sorting every ticket on each read.
CREATE INDEX IF NOT EXISTS idx_tickets_created ON tickets(created_at);
CREATE INDEX IF NOT EXISTS idx_tickets_project_created ON tickets(project_id, created_at);
DROP INDEX IF EXISTS idx_tickets_project;
