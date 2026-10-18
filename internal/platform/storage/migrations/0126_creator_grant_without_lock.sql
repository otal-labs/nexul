-- A doc's creator locks it only through their roles (ADR 0107, amended): docs:lock leaves each creator's own grant.
UPDATE permission_overwrites SET allow = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(permission_overwrites.allow)
        WHERE value <> 'docs:lock'
        ORDER BY value
    )
)
WHERE resource_type = 'doc'
  AND target = 'person'
  AND EXISTS (SELECT 1 FROM json_each(permission_overwrites.allow) WHERE value = 'docs:lock')
  AND EXISTS (SELECT 1 FROM docs d WHERE d.id = permission_overwrites.resource_id AND d.created_by = permission_overwrites.user_id);
