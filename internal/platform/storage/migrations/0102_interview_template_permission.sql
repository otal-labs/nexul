-- Whoever could edit the workspace Interview template through memories:write keeps doing so (ADR 0148).
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'memories:template'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'memories:write');
