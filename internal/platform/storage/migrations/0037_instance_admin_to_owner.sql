-- Instance administration is permission bits now, which an Owner holds by bypass (ADR 0088). An instance whose
-- administrators hold no active Owner role would be locked out, so its earliest active administrator takes the
-- default workspace's Owner role (the owner wizard made it) before the flag goes.
WITH promote AS (
  SELECT u.id FROM users u
  WHERE u.can_create_workspace = 1 AND u.account_status = 'active'
    AND NOT EXISTS (
      SELECT 1 FROM workspace_members m
      JOIN roles r ON r.id = m.role_id AND r.is_owner_role = 1
      JOIN users o ON o.id = m.user_id AND o.account_status = 'active'
    )
  ORDER BY u.created_at, u.id
  LIMIT 1
)
INSERT INTO workspace_members (user_id, workspace_id, role_id, created_at)
SELECT p.id, 'workspace-default', r.id, strftime('%s','now')
FROM promote p JOIN roles r ON r.workspace_id = 'workspace-default' AND r.is_owner_role = 1
WHERE true
ON CONFLICT (user_id, workspace_id) DO UPDATE SET role_id = excluded.role_id;

ALTER TABLE users DROP COLUMN can_create_workspace;
