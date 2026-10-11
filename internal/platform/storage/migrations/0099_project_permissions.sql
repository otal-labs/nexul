-- Creating a project and managing a project's access are their own bits (ADR 0148); whoever could create one through
-- projects:write, or manage people through members:write, keeps doing so.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'projects:create'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'projects:write');

UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'projects:access'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'members:write');
