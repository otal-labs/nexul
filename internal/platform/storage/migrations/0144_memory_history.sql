-- Who a memory version credits (a JSON array of user ids, the first being author_id), each version's size against the kept version before it
-- (null on a memory's oldest kept), and the version a restore copied it from (null for any other save).
-- Kept versions are credited, sized and pruned to the newest 10 per memory once, in Go, right after this script (backfillMemoryHistory).
ALTER TABLE memory_versions ADD COLUMN editor_ids TEXT NOT NULL DEFAULT '[]';
ALTER TABLE memory_versions ADD COLUMN words_added INTEGER;
ALTER TABLE memory_versions ADD COLUMN words_removed INTEGER;
ALTER TABLE memory_versions ADD COLUMN restored_from INTEGER;
