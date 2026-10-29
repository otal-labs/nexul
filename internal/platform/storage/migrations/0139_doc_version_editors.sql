-- Who a version credits (a JSON array of user ids, the first being author_id) and the play or automation it came through; empty for versions written before they were kept.
ALTER TABLE doc_versions ADD COLUMN editor_ids TEXT NOT NULL DEFAULT '[]';
ALTER TABLE doc_versions ADD COLUMN via TEXT NOT NULL DEFAULT '';
