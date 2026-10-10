-- An auto play starts its play when a moment matches (ADR 0132); it goes with its play.
CREATE TABLE IF NOT EXISTS auto_plays (
    id                  TEXT PRIMARY KEY,
    play_id             TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    workspace_id        TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    enabled             INTEGER NOT NULL DEFAULT 0,
    moment              TEXT NOT NULL,
    moment_stage        TEXT,
    conditions          TEXT NOT NULL DEFAULT '{"match":"all","groups":[]}',
    priority            TEXT NOT NULL DEFAULT '{"rules":[],"otherwise":"normal"}',
    once_within_minutes INTEGER NOT NULL DEFAULT 0,
    run_on              TEXT NOT NULL DEFAULT 'developer',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          INTEGER NOT NULL,
    updated_at          INTEGER NOT NULL
);
-- Serves ListAutoPlaysByPlays: each play's auto plays, oldest first.
CREATE INDEX IF NOT EXISTS idx_auto_plays_play ON auto_plays(play_id, created_at, id);
-- Serves the matcher: a workspace's switched-on auto plays for one moment.
CREATE INDEX IF NOT EXISTS idx_auto_plays_moment ON auto_plays(workspace_id, moment) WHERE enabled = 1;

-- The cap on automatic runs per ticket per rolling day, across every auto play of the workspace.
ALTER TABLE workspaces ADD COLUMN auto_play_daily_cap INTEGER NOT NULL DEFAULT 5;

-- Auto plays belong to plays (ADR 0132): read from plays:read, read and write from plays:write, delete from plays:delete.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'autoplays:read' WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value IN ('plays:read', 'plays:write'))
        UNION SELECT 'autoplays:write' WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'plays:delete')
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value IN ('plays:read', 'plays:write', 'plays:delete'));

UPDATE permission_overwrites SET allow = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(permission_overwrites.allow)
        UNION SELECT 'autoplays:read' WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.allow) WHERE value IN ('plays:read', 'plays:write'))
        UNION SELECT 'autoplays:write' WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.allow) WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.allow) WHERE value = 'plays:delete')
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.allow) WHERE value IN ('plays:read', 'plays:write', 'plays:delete'));

-- A deny keeps denying its own verb only: denying plays:write never took away reading.
UPDATE permission_overwrites SET deny = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(permission_overwrites.deny)
        UNION SELECT 'autoplays:read' WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.deny) WHERE value = 'plays:read')
        UNION SELECT 'autoplays:write' WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.deny) WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.deny) WHERE value = 'plays:delete')
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.deny) WHERE value IN ('plays:read', 'plays:write', 'plays:delete'));

UPDATE invitation_grants SET allow_json = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(invitation_grants.allow_json)
        UNION SELECT 'autoplays:read' WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.allow_json) WHERE value IN ('plays:read', 'plays:write'))
        UNION SELECT 'autoplays:write' WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.allow_json) WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.allow_json) WHERE value = 'plays:delete')
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.allow_json) WHERE value IN ('plays:read', 'plays:write', 'plays:delete'));

UPDATE invitation_grants SET deny_json = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(invitation_grants.deny_json)
        UNION SELECT 'autoplays:read' WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.deny_json) WHERE value = 'plays:read')
        UNION SELECT 'autoplays:write' WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.deny_json) WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.deny_json) WHERE value = 'plays:delete')
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.deny_json) WHERE value IN ('plays:read', 'plays:write', 'plays:delete'));

-- An invitation's per-project levels are left alone: they hold project-area bits, and plays are workspace-wide.

UPDATE integration_installs SET scopes = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(integration_installs.scopes)
        UNION SELECT 'autoplays:read' WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value IN ('plays:read', 'plays:write'))
        UNION SELECT 'autoplays:write' WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value = 'plays:delete')
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value IN ('plays:read', 'plays:write', 'plays:delete'));

-- automations.scopes is JSON text in a BLOB, possibly empty; json_each would read a raw BLOB as JSONB.
UPDATE automations SET scopes = CAST((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
        UNION SELECT 'autoplays:read' WHERE EXISTS (
            SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
            WHERE value IN ('plays:read', 'plays:write'))
        UNION SELECT 'autoplays:write' WHERE EXISTS (
            SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
            WHERE value = 'plays:write')
        UNION SELECT 'autoplays:delete' WHERE EXISTS (
            SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
            WHERE value = 'plays:delete')
        ORDER BY value
    )
) AS BLOB)
WHERE EXISTS (
    SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
    WHERE value IN ('plays:read', 'plays:write', 'plays:delete')
);
