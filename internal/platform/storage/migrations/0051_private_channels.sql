-- A private channel is read only by its members, its conversation_participants rows; every existing channel stays public.
ALTER TABLE conversations ADD COLUMN private INTEGER NOT NULL DEFAULT 0;
