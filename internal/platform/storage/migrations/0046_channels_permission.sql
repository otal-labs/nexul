-- Channels are created, renamed, and deleted with their own bits (ADR 0094); whatever could create one through chat:write keeps doing so.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'channels:write'
        UNION SELECT 'channels:delete'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'chat:write');

UPDATE integration_installs SET scopes = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(integration_installs.scopes)
        UNION SELECT 'channels:write'
        UNION SELECT 'channels:delete'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value = 'chat:write');

-- automations.scopes is JSON text in a BLOB, possibly empty; json_each would read a raw BLOB as JSONB.
UPDATE automations SET scopes = CAST((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
        UNION SELECT 'channels:write'
        UNION SELECT 'channels:delete'
        ORDER BY value
    )
) AS BLOB)
WHERE EXISTS (
    SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
    WHERE value = 'chat:write'
);

-- A workspace's #general may be renamed but never deleted, so it is marked rather than recognised by its name.
ALTER TABLE conversations ADD COLUMN is_general INTEGER NOT NULL DEFAULT 0;
UPDATE conversations SET is_general = 1 WHERE kind = 'channel' AND name = 'general';
