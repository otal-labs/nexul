-- A doc names who wrote it; older rows take their earliest overwrite, the creator grant Create writes.
ALTER TABLE docs ADD COLUMN created_by TEXT NOT NULL DEFAULT '';
UPDATE docs SET created_by = COALESCE((
    SELECT po.user_id FROM permission_overwrites po
    WHERE po.resource_type = 'doc' AND po.resource_id = docs.id
    ORDER BY po.created_at, po.user_id
    LIMIT 1
), '');
