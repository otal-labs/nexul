-- Automations belong to a workspace (ADR 0095). Every existing row lands in the default workspace, the only one
-- their permissions were ever checked in.
ALTER TABLE automations ADD COLUMN workspace_id TEXT NOT NULL DEFAULT '';
UPDATE automations SET workspace_id = 'workspace-default';
CREATE INDEX idx_automations_workspace ON automations(workspace_id);

-- Every other workspace gets its own copy of each default with today's switch; config is left empty because its
-- status ids name columns of the default workspace's projects. The code lands at the next start.
INSERT INTO automations (id, name, description, kind, enabled, subscriptions, config_schema, config_values, scopes,
    created_by, token_hash, token_prefix, token_revoked_at, host_id, created_at, updated_at, workspace_id)
SELECT a.id || '-' || w.id, a.name, a.description, a.kind, a.enabled, a.subscriptions, a.config_schema, '{}', a.scopes,
    a.created_by, lower(hex(randomblob(32))), '', NULL, a.host_id, a.created_at, a.updated_at, w.id
FROM automations a CROSS JOIN workspaces w
WHERE a.kind = 'default' AND a.workspace_id = 'workspace-default' AND w.id != 'workspace-default';

-- Secrets are a pool per workspace; the existing pool is the default workspace's.
CREATE TABLE automation_secrets_scoped (
    workspace_id TEXT NOT NULL,
    name         TEXT NOT NULL,
    value        TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    PRIMARY KEY (workspace_id, name)
);
INSERT INTO automation_secrets_scoped (workspace_id, name, value, created_at, updated_at)
SELECT 'workspace-default', name, value, created_at, updated_at FROM automation_secrets;
DROP TABLE automation_secrets;
ALTER TABLE automation_secrets_scoped RENAME TO automation_secrets;

-- The built-in decisions check is switched per workspace (ADR 0066), off everywhere until someone turns it on.
ALTER TABLE workspaces ADD COLUMN decisions_check_enabled INTEGER NOT NULL DEFAULT 0;
