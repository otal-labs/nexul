-- The newest harness turn Nexul saw end on a conversation's agent thread, so a later one there reads as news (ADR 0127).
ALTER TABLE conversations ADD COLUMN agent_seen TEXT NOT NULL DEFAULT '';
-- Serves GetConversationByAgentThread: a harness reports changes by its own thread id.
CREATE INDEX IF NOT EXISTS idx_conversations_agent_thread ON conversations(agent_thread_id) WHERE agent_thread_id != '';
-- Serves LatestPlayTrailInConversation: the run a thread's news reopens is its conversation's newest.
CREATE INDEX IF NOT EXISTS idx_play_trails_conversation ON play_trails(conversation_id, started_at, id);
