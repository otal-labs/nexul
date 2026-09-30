-- Reading container logs is its own verb (ADR 0090); every role that could already change a stack keeps seeing its output.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'stacks:logs'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'stacks:write')
  AND NOT EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'stacks:logs');
