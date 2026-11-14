-- The Owner's own style rules for the instance, applied after its colours; empty means none.
ALTER TABLE instance_branding ADD COLUMN css TEXT NOT NULL DEFAULT '';
