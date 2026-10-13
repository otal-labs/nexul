-- Every reader posts in a channel today and a token posted with chat:write, so channels:post keeps both (ADR 0149).
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        -- The role editor's Post verb sits on the channels ladder, which needs Read to tick.
        UNION SELECT 'channels:read'
        UNION SELECT 'channels:post'
        ORDER BY value
    )
)
WHERE NOT EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'channels:post');

UPDATE integration_installs SET scopes = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(integration_installs.scopes)
        UNION SELECT 'channels:post'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value = 'chat:write');

-- automations.scopes is JSON text in a BLOB, possibly empty; json_each would read a raw BLOB as JSONB.
UPDATE automations SET scopes = CAST((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
        UNION SELECT 'channels:post'
        ORDER BY value
    )
) AS BLOB)
WHERE EXISTS (
    SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
    WHERE value = 'chat:write'
);
