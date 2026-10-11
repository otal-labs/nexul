-- A member holds a set of roles (ADR 0148): one row per role, in the order given, going with the membership and with
-- the role. Every member's one role is copied as a set of one.
CREATE TABLE IF NOT EXISTS member_roles (
    user_id      TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    role_id      TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    PRIMARY KEY (user_id, workspace_id, role_id),
    FOREIGN KEY (user_id, workspace_id) REFERENCES workspace_members(user_id, workspace_id) ON DELETE CASCADE
);
-- Serves a role's delete and CountRoleHolders.
CREATE INDEX IF NOT EXISTS idx_member_roles_role ON member_roles(role_id);
INSERT OR IGNORE INTO member_roles (user_id, workspace_id, role_id, created_at)
SELECT user_id, workspace_id, role_id, created_at FROM workspace_members;

-- Membership stays its own row, now without a role; restricted stays until the switch.
CREATE TABLE workspace_members_new (
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    restricted   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, workspace_id)
);
INSERT INTO workspace_members_new (user_id, workspace_id, created_at, restricted)
SELECT user_id, workspace_id, created_at, restricted FROM workspace_members;
DROP TABLE workspace_members;
ALTER TABLE workspace_members_new RENAME TO workspace_members;
