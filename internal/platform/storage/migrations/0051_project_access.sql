-- A Restricted member sees only the projects they hold Project access to (ADR 0097); every existing member stays From role.
ALTER TABLE workspace_members ADD COLUMN restricted INTEGER NOT NULL DEFAULT 0;

-- An invitation's workspace grant carries Every project and, under None, the levels per project; existing ones stay From role.
ALTER TABLE invitation_grants ADD COLUMN restricted INTEGER NOT NULL DEFAULT 0;
ALTER TABLE invitation_grants ADD COLUMN project_access_json TEXT NOT NULL DEFAULT '[]';
