-- The memories an auto play's runs read beside the ones always included, as a JSON list of ids; each run reads only those
-- in its own project. None for every auto play that exists.
ALTER TABLE auto_plays ADD COLUMN memory_ids TEXT NOT NULL DEFAULT '[]';
