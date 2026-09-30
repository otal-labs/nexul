-- Serves CreateConversation's duplicate check: a text channel name is unique per workspace ignoring case, since names are stored as typed.
DROP INDEX IF EXISTS idx_conversations_channel_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_channel_name ON conversations(workspace_id, name COLLATE NOCASE) WHERE kind = 'channel';
