-- ListConversationsForUser and UnreadCounts, a person's DMs from any workspace (ADR 0141): read from the person's
-- participant rows instead of scanning every conversation.
CREATE INDEX IF NOT EXISTS idx_conversation_participants_user ON conversation_participants(user_id, conversation_id);
