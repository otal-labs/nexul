-- A role's one-line description, shown on invitations and wherever someone picks roles. Existing roles keep none: a
-- workspace's copy of a template role is never rewritten, and a role without one shows its name alone.
ALTER TABLE roles ADD COLUMN description TEXT NOT NULL DEFAULT '';
