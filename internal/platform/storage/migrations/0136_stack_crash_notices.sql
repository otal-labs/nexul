-- The conversation a stack posts its crash notices to, and the bot of it they post as; empty for every stack that exists.
ALTER TABLE stacks ADD COLUMN crash_notice_conversation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE stacks ADD COLUMN crash_notice_bot_id TEXT NOT NULL DEFAULT '';
