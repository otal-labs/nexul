-- Locking a doc is its own bit (ADR 0107); whatever could lock one through docs:write keeps doing so, and a deny on docs:write keeps denying the lock.
UPDATE roles SET permissions = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(roles.permissions)
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(roles.permissions) WHERE value = 'docs:write');

UPDATE permission_overwrites SET allow = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(permission_overwrites.allow)
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.allow) WHERE value = 'docs:write');

UPDATE permission_overwrites SET deny = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(permission_overwrites.deny)
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(permission_overwrites.deny) WHERE value = 'docs:write');

UPDATE invitation_grants SET allow_json = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(invitation_grants.allow_json)
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.allow_json) WHERE value = 'docs:write');

UPDATE invitation_grants SET deny_json = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(invitation_grants.deny_json)
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(invitation_grants.deny_json) WHERE value = 'docs:write');

-- A pending invitation's per-project levels nest an allow list in each element of project_access_json.
UPDATE invitation_grants SET project_access_json = (
    SELECT json_group_array(CASE
        WHEN EXISTS (SELECT 1 FROM json_each(pa.value, '$.allow') WHERE value = 'docs:write') THEN json_set(json(pa.value), '$.allow', json((
            SELECT json_group_array(value) FROM (
                SELECT value FROM json_each(pa.value, '$.allow')
                UNION SELECT 'docs:lock'
                ORDER BY value
            )
        )))
        ELSE json(pa.value)
    END)
    FROM json_each(invitation_grants.project_access_json) AS pa
)
WHERE EXISTS (
    SELECT 1 FROM json_each(invitation_grants.project_access_json) AS pa, json_each(pa.value, '$.allow') AS a
    WHERE a.value = 'docs:write'
);

UPDATE integration_installs SET scopes = (
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(integration_installs.scopes)
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
)
WHERE EXISTS (SELECT 1 FROM json_each(integration_installs.scopes) WHERE value = 'docs:write');

-- automations.scopes is JSON text in a BLOB, possibly empty; json_each would read a raw BLOB as JSONB.
UPDATE automations SET scopes = CAST((
    SELECT json_group_array(value) FROM (
        SELECT value FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
        UNION SELECT 'docs:lock'
        ORDER BY value
    )
) AS BLOB)
WHERE EXISTS (
    SELECT 1 FROM json_each(iif(json_valid(CAST(automations.scopes AS TEXT)), CAST(automations.scopes AS TEXT), '[]'))
    WHERE value = 'docs:write'
);
