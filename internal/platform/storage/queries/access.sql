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
