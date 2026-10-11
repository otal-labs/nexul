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
SELECT * FROM permission_overwrites WHERE user_id = ? AND resource_type IN ('workspace', 'project');

-- name: ListOwnOverwritesInProject :many
SELECT po.* FROM permission_overwrites po
WHERE po.user_id = sqlc.arg(user_id) AND po.resource_type = 'workspace' AND po.resource_id = sqlc.arg(workspace_id)
UNION ALL
SELECT pp.* FROM permission_overwrites pp
WHERE pp.user_id = sqlc.arg(user_id) AND pp.resource_type = 'project' AND pp.resource_id = sqlc.arg(project_id);

-- name: ListRoleOverwrites :many
SELECT * FROM permission_overwrites
WHERE role_id IN (SELECT value FROM json_each(sqlc.arg(role_ids))) AND resource_type = sqlc.arg(resource_type);

-- name: ListRoleOverwritesInResource :many
SELECT * FROM permission_overwrites
WHERE resource_type = sqlc.arg(resource_type) AND resource_id = sqlc.arg(resource_id) AND target = 'role'
  AND role_id IN (SELECT value FROM json_each(sqlc.arg(role_ids)))
ORDER BY role_id;

-- name: ListDocRuleTurns :many
SELECT rs.resource_id, rs.state FROM (
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
) rs JOIN docs d ON d.id = rs.resource_id
WHERE (rs.state = 0 AND d.project_id IN (SELECT value FROM json_each(sqlc.arg(readable))))
   OR (rs.state = 1 AND d.project_id NOT IN (SELECT value FROM json_each(sqlc.arg(readable))))
ORDER BY rs.resource_id;
