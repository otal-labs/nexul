-- memory_ids is a JSON list, so no foreign key drops a deleted memory from it; this trigger does, and the update clears ids already left behind.
CREATE TRIGGER IF NOT EXISTS memories_auto_plays_ad AFTER DELETE ON memories BEGIN
    UPDATE auto_plays SET memory_ids = (SELECT json_group_array(value) FROM json_each(auto_plays.memory_ids) WHERE value != old.id)
    WHERE EXISTS (SELECT 1 FROM json_each(auto_plays.memory_ids) WHERE value = old.id);
END;

UPDATE auto_plays SET memory_ids = (SELECT json_group_array(value) FROM json_each(auto_plays.memory_ids) WHERE value IN (SELECT id FROM memories))
WHERE EXISTS (SELECT 1 FROM json_each(auto_plays.memory_ids) WHERE value NOT IN (SELECT id FROM memories));
