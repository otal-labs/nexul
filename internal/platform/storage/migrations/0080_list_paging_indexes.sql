-- Each list tool reads one page in SQL (ADR 0140); each index below walks one list's order to its page instead of sorting
-- every row, and ends in id, the tiebreaker that keeps an offset from skipping or repeating rows that share a second.

-- dead_letter_list, newest first.
CREATE INDEX IF NOT EXISTS idx_dead_letters_created ON dead_letters(created_at, id);

-- message_list, a conversation's live messages newest first, and their count, both from the index alone.
CREATE INDEX IF NOT EXISTS idx_messages_live ON messages(conversation_id, created_at, id) WHERE deleted_at IS NULL;

-- notification_list, one person's inbox newest first. It replaces the (user_id, created_at) index, which left the
-- id tiebreak to a sort of the whole inbox.
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at, id);
DROP INDEX IF EXISTS idx_notifications_user;

-- deploy_list, newest first: across every stack, one stack's history, and one status. The two with a leading column
-- replace the single-column indexes on it, whose lookups they still serve.
CREATE INDEX IF NOT EXISTS idx_deploys_created ON deploys(created_at, id);
CREATE INDEX IF NOT EXISTS idx_deploys_stack_created ON deploys(stack_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_deploys_status_created ON deploys(status, created_at, id);
DROP INDEX IF EXISTS idx_deploys_stack_id;
DROP INDEX IF EXISTS idx_deploys_status;

-- ticket_list, oldest first: across every project, one project, and one doc's tickets. Each replaces an index without
-- the id tiebreak, whose lookups it still serves.
CREATE INDEX IF NOT EXISTS idx_tickets_created_id ON tickets(created_at, id);
CREATE INDEX IF NOT EXISTS idx_tickets_project_created_id ON tickets(project_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_tickets_doc_created ON tickets(doc_id, created_at, id);
DROP INDEX IF EXISTS idx_tickets_created;
DROP INDEX IF EXISTS idx_tickets_project_created;
DROP INDEX IF EXISTS idx_tickets_doc_id;
