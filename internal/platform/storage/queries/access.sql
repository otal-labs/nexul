-- name: GetOverwrite :one
SELECT * FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? AND user_id = ?;

-- name: ListOverwritesByResource :many
SELECT * FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? ORDER BY target, coalesce(user_id, role_id);

-- name: UpsertOverwrite :execrows
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at)
VALUES (sqlc.arg(resource_type), sqlc.arg(resource_id), 'person', sqlc.arg(user_id), sqlc.arg(allow), sqlc.arg(deny), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT(resource_type, resource_id, target, coalesce(user_id, role_id)) DO UPDATE SET
  allow = excluded.allow, deny = excluded.deny, updated_at = excluded.updated_at;

-- name: DeleteOverwrite :execrows
DELETE FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? AND user_id = ?;

-- name: GetRoleOverwrite :one
SELECT * FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? AND target = 'role' AND role_id = ?;

-- name: UpsertRoleOverwrite :execrows
INSERT INTO permission_overwrites (resource_type, resource_id, target, role_id, allow, deny, created_at, updated_at)
VALUES (sqlc.arg(resource_type), sqlc.arg(resource_id), 'role', sqlc.arg(role_id), sqlc.arg(allow), sqlc.arg(deny), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT(resource_type, resource_id, target, coalesce(user_id, role_id)) DO UPDATE SET
  allow = excluded.allow, deny = excluded.deny, updated_at = excluded.updated_at;

-- name: DeleteRoleOverwrite :execrows
DELETE FROM permission_overwrites WHERE resource_type = ? AND resource_id = ? AND target = 'role' AND role_id = ?;

-- name: DeleteOverwritesByResource :exec
DELETE FROM permission_overwrites WHERE resource_type = ? AND resource_id = ?;

-- name: CountOverwriteAllowAny :one
SELECT COUNT(*) FROM (
  SELECT 1 FROM permission_overwrites po
  WHERE po.user_id = sqlc.arg(user_id) AND po.resource_type = sqlc.arg(resource_type) AND instr(po.allow, sqlc.arg(quoted_action)) > 0
  UNION ALL
  SELECT 1 FROM permission_overwrites pr
  WHERE pr.role_id IN (SELECT mr.role_id FROM member_roles mr WHERE mr.user_id = sqlc.arg(user_id))
    AND pr.resource_type = sqlc.arg(resource_type) AND instr(pr.allow, sqlc.arg(quoted_action)) > 0
);

-- name: ListOverwriteRules :many
SELECT po.* FROM permission_overwrites po
WHERE po.resource_type = sqlc.arg(resource_type)
  AND (po.user_id = sqlc.arg(user_id) OR po.role_id IN (SELECT mr.role_id FROM member_roles mr WHERE mr.user_id = sqlc.arg(user_id)))
  AND po.resource_id IN (sqlc.slice('resource_ids'));

-- name: ListOverwriteRulesOfType :many
SELECT po.* FROM permission_overwrites po WHERE po.user_id = sqlc.arg(user_id) AND po.resource_type = sqlc.arg(resource_type)
UNION ALL
SELECT pr.* FROM permission_overwrites pr
WHERE pr.role_id IN (SELECT mr.role_id FROM member_roles mr WHERE mr.user_id = sqlc.arg(user_id)) AND pr.resource_type = sqlc.arg(resource_type);

-- name: ListOwnOverwrites :many
SELECT * FROM permission_overwrites WHERE user_id = ? AND resource_type = 'project';

-- name: ListOwnOverwritesInProject :many
SELECT * FROM permission_overwrites WHERE user_id = ? AND resource_type = 'project' AND resource_id = ?;

-- name: ListRoleOverwrites :many
SELECT * FROM permission_overwrites
WHERE role_id IN (SELECT value FROM json_each(sqlc.arg(role_ids))) AND resource_type = sqlc.arg(resource_type);

-- name: ListRoleOverwritesInResource :many
SELECT * FROM permission_overwrites
WHERE resource_type = sqlc.arg(resource_type) AND resource_id = sqlc.arg(resource_id) AND target = 'role'
  AND role_id IN (SELECT value FROM json_each(sqlc.arg(role_ids)))
ORDER BY role_id;

-- name: ListDocChainRules :many
SELECT po.resource_id AS doc_id, po.* FROM permission_overwrites po
WHERE po.resource_type = 'doc' AND po.resource_id IN (SELECT value FROM json_each(sqlc.arg(doc_ids)))
  AND (po.user_id = sqlc.arg(user_id) OR po.role_id IN (SELECT mr.role_id FROM member_roles mr WHERE mr.user_id = sqlc.arg(user_id)))
UNION ALL
SELECT d.id AS doc_id, pf.* FROM docs d JOIN permission_overwrites pf ON pf.resource_type = 'folder' AND pf.resource_id = d.folder_id
WHERE d.id IN (SELECT value FROM json_each(sqlc.arg(doc_ids)))
  AND (pf.user_id = sqlc.arg(user_id) OR pf.role_id IN (SELECT mr.role_id FROM member_roles mr WHERE mr.user_id = sqlc.arg(user_id)));

-- name: ListDocRuleTurns :many
WITH folder_states AS (
  SELECT u.resource_id, COALESCE(MAX(CASE WHEN u.by_role = 0 THEN u.state END), MAX(CASE WHEN u.by_role = 1 THEN u.state END)) AS state
  FROM (
    SELECT po.resource_id, 0 AS by_role,
      CASE WHEN instr(po.deny, sqlc.arg(quoted_action)) > 0 THEN 0 WHEN instr(po.allow, sqlc.arg(quoted_action)) > 0 THEN 1 END AS state
    FROM permission_overwrites po JOIN doc_folders rf ON rf.id = po.resource_id
    WHERE po.user_id = sqlc.arg(user_id) AND po.resource_type = 'folder' AND po.target = 'person'
      AND rf.project_id IN (SELECT value FROM json_each(sqlc.arg(ruled)))
    UNION ALL
    SELECT pr.resource_id, 1 AS by_role,
      CASE WHEN instr(pr.deny, sqlc.arg(quoted_action)) > 0 THEN 0 WHEN instr(pr.allow, sqlc.arg(quoted_action)) > 0 THEN 1 END AS state
    FROM member_roles mr CROSS JOIN permission_overwrites pr CROSS JOIN doc_folders rf CROSS JOIN projects rp
    WHERE mr.user_id = sqlc.arg(user_id) AND pr.role_id = mr.role_id AND pr.resource_type = 'folder' AND rf.id = pr.resource_id
      AND rp.id = rf.project_id AND rp.workspace_id = mr.workspace_id
      AND rf.project_id IN (SELECT value FROM json_each(sqlc.arg(ruled)))
  ) u
  GROUP BY u.resource_id
), doc_states AS (
  SELECT u.resource_id, COALESCE(MAX(CASE WHEN u.by_role = 0 THEN u.state END), MAX(CASE WHEN u.by_role = 1 THEN u.state END)) AS state
  FROM (
    SELECT po.resource_id, 0 AS by_role,
      CASE WHEN instr(po.deny, sqlc.arg(quoted_action)) > 0 THEN 0 WHEN instr(po.allow, sqlc.arg(quoted_action)) > 0 THEN 1 END AS state
    FROM permission_overwrites po JOIN docs rd ON rd.id = po.resource_id
    WHERE po.user_id = sqlc.arg(user_id) AND po.resource_type = 'doc' AND po.target = 'person'
      AND rd.project_id IN (SELECT value FROM json_each(sqlc.arg(ruled)))
    UNION ALL
    SELECT pr.resource_id, 1 AS by_role,
      CASE WHEN instr(pr.deny, sqlc.arg(quoted_action)) > 0 THEN 0 WHEN instr(pr.allow, sqlc.arg(quoted_action)) > 0 THEN 1 END AS state
    FROM member_roles mr CROSS JOIN permission_overwrites pr CROSS JOIN docs rd CROSS JOIN projects rp
    WHERE mr.user_id = sqlc.arg(user_id) AND pr.role_id = mr.role_id AND pr.resource_type = 'doc' AND rd.id = pr.resource_id
      AND rp.id = rd.project_id AND rp.workspace_id = mr.workspace_id
      AND rd.project_id IN (SELECT value FROM json_each(sqlc.arg(ruled)))
  ) u
  GROUP BY u.resource_id
)
SELECT 'folder' AS kind, fs.resource_id, fs.state FROM folder_states fs JOIN doc_folders f ON f.id = fs.resource_id
WHERE (fs.state = 0 AND f.project_id IN (SELECT value FROM json_each(sqlc.arg(readable))))
   OR (fs.state = 1 AND f.project_id NOT IN (SELECT value FROM json_each(sqlc.arg(readable))))
UNION ALL
SELECT 'doc' AS kind, ds.resource_id, ds.state FROM doc_states ds JOIN docs d ON d.id = ds.resource_id
LEFT JOIN folder_states fs ON fs.resource_id = d.folder_id
WHERE ds.state != COALESCE(fs.state, d.project_id IN (SELECT value FROM json_each(sqlc.arg(readable))))
ORDER BY kind, resource_id;

-- name: GetAccessSetup :one
SELECT w.access_setup_finished, w.access_setup_by, w.access_setup_at, COALESCE(u.login, '') AS finished_by_login
FROM workspaces w LEFT JOIN users u ON u.id = w.access_setup_by
WHERE w.id = ?;

-- name: ListAccessGridProjects :many
SELECT id, name, prefix FROM projects WHERE workspace_id = ? ORDER BY position, id;

-- name: ListAccessGridRoles :many
SELECT r.id, r.name, (SELECT COUNT(*) FROM member_roles mr WHERE mr.role_id = r.id) AS holders
FROM roles r WHERE r.workspace_id = ? AND r.is_owner_role = 0 ORDER BY r.created_at, r.id;

-- name: ListAccessGridChannels :many
SELECT id, name, kind FROM conversations
WHERE workspace_id = ? AND kind IN ('channel', 'voice_channel') AND private = 0
ORDER BY is_general DESC, kind, name, id;

-- name: ListAccessGridMembers :many
SELECT user_id FROM workspace_members WHERE workspace_id = ?;

-- name: ListAccessGridRules :many
SELECT po.* FROM permission_overwrites po JOIN projects p ON p.id = po.resource_id
WHERE po.resource_type = 'project' AND p.workspace_id = sqlc.arg(workspace_id)
UNION ALL
SELECT co.* FROM permission_overwrites co JOIN conversations c ON c.id = co.resource_id
WHERE co.resource_type = 'channel' AND c.workspace_id = sqlc.arg(workspace_id);

-- name: UpsertOverwritesFromJSON :exec
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at)
SELECT json_extract(j.value, '$.resource_type'), json_extract(j.value, '$.resource_id'), json_extract(j.value, '$.target'),
       json_extract(j.value, '$.user_id'), json_extract(j.value, '$.role_id'), json_extract(j.value, '$.allow'),
       json_extract(j.value, '$.deny'), sqlc.arg(now), sqlc.arg(now)
FROM json_each(sqlc.arg(rules)) j WHERE true
ON CONFLICT(resource_type, resource_id, target, coalesce(user_id, role_id)) DO UPDATE SET
  allow = excluded.allow, deny = excluded.deny, updated_at = excluded.updated_at;

-- name: DeleteOverwritesFromJSON :exec
DELETE FROM permission_overwrites WHERE rowid IN (
  SELECT po.rowid FROM json_each(sqlc.arg(rules)) j JOIN permission_overwrites po
    ON po.resource_type = json_extract(j.value, '$.resource_type') AND po.resource_id = json_extract(j.value, '$.resource_id')
   AND po.target = json_extract(j.value, '$.target')
   AND coalesce(po.user_id, po.role_id) = coalesce(json_extract(j.value, '$.user_id'), json_extract(j.value, '$.role_id'))
);

-- name: FinishAccessSetup :exec
UPDATE workspaces SET access_setup_finished = 1, access_setup_by = ?, access_setup_at = ? WHERE id = ?;
