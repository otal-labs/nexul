-- A version a restore wrote names the version number it restored from; null for every other version, and for restores made before this.
-- Each version's size against the kept version before it: words added and removed. Null on an item's oldest kept version.
-- The sizes of versions already kept are filled once, in Go, right after this script (backfillVersionSizes).
ALTER TABLE doc_versions ADD COLUMN words_added INTEGER;
ALTER TABLE doc_versions ADD COLUMN words_removed INTEGER;
ALTER TABLE ticket_description_versions ADD COLUMN words_added INTEGER;
ALTER TABLE ticket_description_versions ADD COLUMN words_removed INTEGER;
ALTER TABLE play_instruction_versions ADD COLUMN words_added INTEGER;
ALTER TABLE play_instruction_versions ADD COLUMN words_removed INTEGER;
ALTER TABLE doc_versions ADD COLUMN restored_from INTEGER;
ALTER TABLE ticket_description_versions ADD COLUMN restored_from INTEGER;
ALTER TABLE play_instruction_versions ADD COLUMN restored_from INTEGER;
