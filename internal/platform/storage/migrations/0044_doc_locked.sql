-- A locked doc refuses edits to its title and body until unlocked; every existing doc starts unlocked.
ALTER TABLE docs ADD COLUMN locked INTEGER NOT NULL DEFAULT 0;
