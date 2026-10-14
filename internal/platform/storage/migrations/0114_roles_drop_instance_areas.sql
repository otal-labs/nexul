-- Only the instance Owner holds the instance by right (ADR 0151), so roles drop the bits that reach it through any
-- workspace: DNS, accounts, instance settings and upgrades, events, templates, adding machines, integrations, the
-- audit log and connectors. Topology and automations stay: on a role they act on that workspace's own canvas and
-- automations. Anyone else gets an area back through a rule on it.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        WHERE value NOT IN ('dns:read', 'dns:write', 'dns:delete', 'accounts:read', 'accounts:write', 'accounts:delete',
            'instance:read', 'instance:write', 'events:read', 'templates:read', 'templates:write', 'machines:create',
            'integrations:read', 'integrations:write', 'integrations:delete', 'audit:read', 'connectors:read',
            'connectors:write')
        ORDER BY value
    )
)
WHERE EXISTS (
    SELECT 1 FROM json_each(roles.permissions)
    WHERE value IN ('dns:read', 'dns:write', 'dns:delete', 'accounts:read', 'accounts:write', 'accounts:delete',
        'instance:read', 'instance:write', 'events:read', 'templates:read', 'templates:write', 'machines:create',
        'integrations:read', 'integrations:write', 'integrations:delete', 'audit:read', 'connectors:read',
        'connectors:write')
);

-- An edited Template role, stored as JSON with its permissions in an array.
UPDATE instance_templates SET body = json_set(body, '$.permissions', json((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(instance_templates.body, '$.permissions')
        WHERE value NOT IN ('dns:read', 'dns:write', 'dns:delete', 'accounts:read', 'accounts:write', 'accounts:delete',
            'instance:read', 'instance:write', 'events:read', 'templates:read', 'templates:write', 'machines:create',
            'integrations:read', 'integrations:write', 'integrations:delete', 'audit:read', 'connectors:read',
            'connectors:write')
        ORDER BY value
    )
)))
WHERE kind = 'role' AND json_valid(body) AND EXISTS (
    SELECT 1 FROM json_each(instance_templates.body, '$.permissions')
    WHERE value IN ('dns:read', 'dns:write', 'dns:delete', 'accounts:read', 'accounts:write', 'accounts:delete',
        'instance:read', 'instance:write', 'events:read', 'templates:read', 'templates:write', 'machines:create',
        'integrations:read', 'integrations:write', 'integrations:delete', 'audit:read', 'connectors:read',
        'connectors:write')
);
