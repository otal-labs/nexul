-- A chosen-computer auto play, and the queued run it makes, name an ordered list of computers (ADR 0165): a JSON list, first
-- ready first. computer_id stays written as the list's first, so a downgrade still reads it.
ALTER TABLE auto_plays ADD COLUMN computer_ids TEXT NOT NULL DEFAULT '[]';
ALTER TABLE play_queue ADD COLUMN computer_ids TEXT NOT NULL DEFAULT '[]';

UPDATE auto_plays SET computer_ids = json_array(computer_id) WHERE computer_id != '';
UPDATE play_queue SET computer_ids = json_array(computer_id) WHERE computer_id != '';
