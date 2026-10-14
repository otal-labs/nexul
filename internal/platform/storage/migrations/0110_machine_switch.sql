-- Seeing a machine is held through its rules alone (ADR 0151), so the bits that meant "every machine or runner" go:
-- runners:read, runners:write, runners:delete and machines:read. Every machine starts with only the instance Owner on
-- it; the instance Owner writes the rules that bring people back.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        WHERE value NOT IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
        ORDER BY value
    )
)
WHERE EXISTS (
    SELECT 1 FROM json_each(roles.permissions)
    WHERE value IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
);

-- A Template role is stored once edited, as JSON with its permissions in an array.
UPDATE instance_templates SET body = json_set(body, '$.permissions', json((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(instance_templates.body, '$.permissions')
        WHERE value NOT IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
        ORDER BY value
    )
)))
WHERE kind = 'role' AND json_valid(body) AND EXISTS (
    SELECT 1 FROM json_each(instance_templates.body, '$.permissions')
    WHERE value IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
);

UPDATE integration_installs SET scopes = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(integration_installs.scopes)
        WHERE value NOT IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
        ORDER BY value
    )
)
WHERE EXISTS (
    SELECT 1 FROM json_each(integration_installs.scopes)
    WHERE value IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
);

-- automations.scopes is JSON text in a BLOB, possibly empty; json_each would read a raw BLOB as JSONB.
UPDATE automations SET scopes = CAST((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
        WHERE value NOT IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
        ORDER BY value
    )
) AS BLOB)
WHERE EXISTS (
    SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
    WHERE value IN ('runners:read', 'runners:write', 'runners:delete', 'machines:read')
);

-- The "runners" instance area becomes "machines" and hands out machines:create alone; the rules on it keep only that
-- action, and one left holding nothing is dropped.
UPDATE permission_overwrites SET
    allow = (SELECT json_group_array(value) FROM (SELECT value FROM json_each(permission_overwrites.allow) WHERE value = 'machines:create' ORDER BY value)),
    deny = (SELECT json_group_array(value) FROM (SELECT value FROM json_each(permission_overwrites.deny) WHERE value = 'machines:create' ORDER BY value))
WHERE resource_type = 'instance_area' AND resource_id = 'runners';
DELETE FROM permission_overwrites WHERE resource_type = 'instance_area' AND resource_id = 'runners' AND allow = '[]' AND deny = '[]';
UPDATE permission_overwrites SET resource_id = 'machines' WHERE resource_type = 'instance_area' AND resource_id = 'runners';
