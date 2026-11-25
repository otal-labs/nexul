-- The one-line preview a doc list shows, written with each body, so a list reads it instead of every full body. The docs
-- already stored get theirs in Go after this script, from the same builder a save uses.
ALTER TABLE docs ADD COLUMN snippet TEXT NOT NULL DEFAULT '';
