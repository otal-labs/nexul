-- The work an Agent reply handed to other agents, as JSON (ADR 0116); NULL on every other message and once deleted.
ALTER TABLE messages ADD COLUMN handoffs TEXT;
