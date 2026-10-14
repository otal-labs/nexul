package storage

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const creatorGrantJSON = `["docs:delete","docs:lock","docs:read","docs:write","permissions:write"]`

// TestMigration_TheWipe upgrades a workspace holding every per-person shape the previous schema could store, and pins
// exactly what survives: roles, who holds them, open links with their role, and each doc creator's own row as stored.
func TestMigration_TheWipe(t *testing.T) {
	db := migrateBefore(t, "0100")
	future := time.Now().Add(24 * time.Hour).Unix()
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES
    ('alice', 'alice', 1, 1), ('bob', 'bob', 1, 1), ('sam', 'sam', 1, 1), ('lena', 'lena', 1, 1), ('nina', 'nina', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES
    ('ws-acme', 'Acme', 'acme', 1, 1), ('ws-quiet', 'Quiet', 'quiet', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES
    ('p-web', 'Web', 'WEB', 1, 'ws-acme', 1, 1), ('p-api', 'API', 'API', 2, 'ws-acme', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'ws-acme', 'Owner', 1, 1, 1, '[]'),
    ('role-dev', 'ws-acme', 'Developer', 0, 2, 2, '["docs:read","docs:write","tickets:read","tickets:write"]'),
    ('role-client', 'ws-acme', 'Client', 0, 3, 3, '["docs:read","tickets:read"]'),
    ('role-quiet-owner', 'ws-quiet', 'Owner', 1, 1, 1, '[]'),
    ('role-quiet-member', 'ws-quiet', 'Member', 0, 2, 2, '["docs:read"]');
INSERT INTO workspace_members (user_id, workspace_id, created_at, restricted) VALUES
    ('alice', 'ws-acme', 10, 0), ('bob', 'ws-acme', 11, 0), ('sam', 'ws-acme', 12, 1), ('lena', 'ws-acme', 13, 0),
    ('alice', 'ws-quiet', 14, 0);
INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES
    ('alice', 'ws-acme', 'role-owner', 10), ('bob', 'ws-acme', 'role-dev', 11), ('sam', 'ws-acme', 'role-client', 12),
    ('lena', 'ws-acme', 'role-dev', 13), ('alice', 'ws-quiet', 'role-quiet-owner', 14);
INSERT INTO docs (id, title, body, created_at, updated_at, project_id, created_by) VALUES
    ('d-spec', 'Spec', '', 1, 1, 'p-web', 'lena'),
    ('d-plan', 'Plan', '', 1, 1, 'p-web', 'bob'),
    ('d-brief', 'Brief', '', 1, 1, 'p-api', 'alice'),
    ('d-left', 'Left behind', '', 1, 1, 'p-api', 'nina');
INSERT INTO plays (id, workspace_id, label, type, created_at, updated_at) VALUES ('play-ship', 'ws-acme', 'Ship it', 'ticket', 1, 1);
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, role_id, allow, deny, created_at, updated_at) VALUES
    ('workspace', 'ws-acme', 'person', 'bob', NULL, '["deploys:write"]', '["tickets:write"]', 1, 1),
    ('workspace', 'ws-acme', 'person', 'alice', NULL, '[]', '["docs:write"]', 1, 1),
    ('project', 'p-web', 'person', 'sam', NULL, '["tickets:read"]', '[]', 1, 1),
    ('project', 'p-api', 'person', 'bob', NULL, '["tickets:read","tickets:write"]', '[]', 1, 1),
    ('project', 'p-web', 'role', NULL, 'role-dev', '["projects:open"]', '[]', 1, 1),
    ('doc', 'd-spec', 'person', 'lena', NULL, '` + creatorGrantJSON + `', '[]', 1, 1),
    ('doc', 'd-spec', 'person', 'sam', NULL, '["docs:read"]', '[]', 1, 1),
    ('doc', 'd-spec', 'role', NULL, 'role-client', '[]', '["docs:read"]', 1, 1),
    ('doc', 'd-plan', 'person', 'bob', NULL, '["docs:delete","docs:lock","docs:read","docs:write","permissions:write","docs:clone"]', '["docs:delete"]', 1, 1),
    ('doc', 'd-plan', 'person', 'lena', NULL, '["docs:read","docs:write"]', '[]', 1, 1),
    ('doc', 'd-brief', 'person', 'alice', NULL, '` + creatorGrantJSON + `', '[]', 1, 1),
    ('doc', 'd-left', 'person', 'nina', NULL, '` + creatorGrantJSON + `', '[]', 1, 1),
    ('doc', 'd-gone', 'person', 'lena', NULL, '` + creatorGrantJSON + `', '[]', 1, 1),
    ('play', 'play-ship', 'person', 'sam', NULL, '[]', '["plays:run"]', 1, 1),
    ('play', 'play-ship', 'person', 'bob', NULL, '["plays:run"]', '[]', 1, 1);
INSERT INTO invitations (id, token_hash, invited_by, created_at, expires_at) VALUES
    ('inv-plain', 'hash-plain', 'alice', 1, ` + itoa(future) + `),
    ('inv-overrides', 'hash-overrides', 'alice', 1, ` + itoa(future) + `),
    ('inv-restricted', 'hash-restricted', 'alice', 1, ` + itoa(future) + `),
    ('inv-two', 'hash-two', 'alice', 1, ` + itoa(future) + `);
INSERT INTO invitation_grants (invitation_id, workspace_id, role_id, allow_json, deny_json, restricted, project_access_json) VALUES
    ('inv-plain', 'ws-acme', 'role-dev', '[]', '[]', 0, '[]'),
    ('inv-overrides', 'ws-acme', 'role-client', '["tickets:write"]', '["docs:read"]', 0, '[]'),
    ('inv-restricted', 'ws-acme', 'role-client', '[]', '[]', 1, '[{"project_id":"p-web","allow":["tickets:read"]}]'),
    ('inv-two', 'ws-acme', 'role-dev', '[]', '["tickets:write"]', 1, '[]'),
    ('inv-two', 'ws-quiet', 'role-quiet-member', '["chat:write"]', '[]', 0, '[]');
INSERT INTO workspace_invites (workspace_id, login, role_id, invited_by, created_at) VALUES ('ws-acme', 'max', 'role-client', 'alice', 1);
`)
	require.NoError(t, err)
	memberships := lines(t, db, `SELECT user_id || '|' || workspace_id || '|' || created_at FROM workspace_members ORDER BY 1`)
	held := lines(t, db, `SELECT user_id || '|' || workspace_id || '|' || role_id || '|' || created_at FROM member_roles ORDER BY 1`)
	roles := lines(t, db, `SELECT id || '|' || permissions FROM roles ORDER BY 1`)

	migrateUpTo(t, db, "0107") // 0100 and every later migration up to 0107, which gives every role channels:post.

	assert.Equal(t, []string{
		"doc|d-brief|person|alice|" + creatorGrantJSON + "|[]",
		`doc|d-left|person|nina|` + creatorGrantJSON + `|[]`,
		`doc|d-plan|person|bob|["docs:delete","docs:lock","docs:read","docs:write","permissions:write","docs:clone"]|["docs:delete"]`,
		"doc|d-spec|person|lena|" + creatorGrantJSON + "|[]",
	}, lines(t, db, `SELECT resource_type || '|' || resource_id || '|' || target || '|' || coalesce(user_id, role_id) || '|' || allow || '|' || deny
FROM permission_overwrites ORDER BY 1`), "each creator's own row, a share or deny in it included, as stored; nothing else")
	assert.Equal(t, memberships, lines(t, db, `SELECT user_id || '|' || workspace_id || '|' || created_at FROM workspace_members ORDER BY 1`))
	assert.Equal(t, held, lines(t, db, `SELECT user_id || '|' || workspace_id || '|' || role_id || '|' || created_at FROM member_roles ORDER BY 1`), "who holds which role")
	assert.Equal(t, roles, lines(t, db, `SELECT id || '|' || permissions FROM roles ORDER BY 1`), "roles are untouched")
	assert.Empty(t, lines(t, db, `SELECT name FROM pragma_table_info('workspace_members') WHERE name = 'restricted'`))

	assert.Equal(t, []string{"invitation_id", "workspace_id", "role_id"}, lines(t, db, `SELECT name FROM pragma_table_info('invitation_grants') ORDER BY cid`))
	assert.Equal(t, []string{
		"inv-overrides|ws-acme|role-client",
		"inv-plain|ws-acme|role-dev",
		"inv-restricted|ws-acme|role-client",
		"inv-two|ws-acme|role-dev",
		"inv-two|ws-quiet|role-quiet-member",
	}, lines(t, db, `SELECT invitation_id || '|' || workspace_id || '|' || role_id FROM invitation_grants ORDER BY 1`), "every open link keeps its role")
	assert.Equal(t, []string{"inv-overrides", "inv-plain", "inv-restricted", "inv-two"}, lines(t, db, `SELECT id FROM invitations ORDER BY 1`))
	assert.Equal(t, []string{"ws-acme|max|role-client"}, lines(t, db, `SELECT workspace_id || '|' || login || '|' || role_id FROM workspace_invites`))

	assert.Equal(t, []string{"workspace-default|1", "ws-acme|0", "ws-quiet|1"},
		lines(t, db, `SELECT id || '|' || access_setup_finished FROM workspaces ORDER BY 1`), "a workspace with projects waits for its setup")

	withArchivedAt(t, db)
	s := New(db, testEncKey)
	ctx := t.Context()
	owner, err := s.WorkspaceMembers.MemberRole(ctx, "ws-acme", "alice")
	require.NoError(t, err)
	assert.True(t, owner.IsOwnerRole, "the Owner stays the Owner")
	client, err := s.WorkspaceMembers.Member(ctx, "ws-acme", "sam")
	require.NoError(t, err)
	assert.Equal(t, []string{"role-client"}, client.RoleIDs, "a former Restricted member keeps their role")
	open, err := s.Invitations.List(ctx, "alice", time.Now())
	require.NoError(t, err)
	shapes := map[string][]string{}
	for _, inv := range open {
		for _, g := range inv.Grants {
			assert.Empty(t, g.Allow, inv.ID)
			assert.Empty(t, g.Deny, inv.ID)
			assert.Empty(t, g.EveryProject, inv.ID)
			assert.Empty(t, g.ProjectAccess, inv.ID)
			shapes[inv.ID] = append(shapes[inv.ID], g.WorkspaceID+"|"+g.RoleID)
		}
	}
	assert.Equal(t, map[string][]string{
		"inv-plain": {"ws-acme|role-dev"}, "inv-overrides": {"ws-acme|role-client"}, "inv-restricted": {"ws-acme|role-client"},
		"inv-two": {"ws-acme|role-dev", "ws-quiet|role-quiet-member"},
	}, shapes, "open links read as their role alone")
}

// TestMigration_TheWipe_RunsOnce: migrating an upgraded database again changes nothing.
func TestMigration_TheWipe_RunsOnce(t *testing.T) {
	db := migrateBefore(t, "0100")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('lena', 'lena', 1, 1);
INSERT INTO projects (id, name, position, workspace_id, created_at, updated_at) VALUES ('p-web', 'Web', 1, 'workspace-default', 1, 1);
INSERT INTO docs (id, title, body, created_at, updated_at, project_id, created_by) VALUES ('d-spec', 'Spec', '', 1, 1, 'p-web', 'lena');
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at) VALUES
    ('doc', 'd-spec', 'person', 'lena', '` + creatorGrantJSON + `', '[]', 1, 1);
`)
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	snapshot := func() []string {
		return append(append(lines(t, db, `SELECT resource_id || '|' || user_id || '|' || allow FROM permission_overwrites ORDER BY 1`),
			lines(t, db, `SELECT id || '|' || access_setup_finished FROM workspaces ORDER BY 1`)...),
			lines(t, db, `SELECT version FROM schema_migrations ORDER BY 1`)...)
	}
	before := snapshot()

	require.NoError(t, Migrate(db))

	assert.Equal(t, before, snapshot())
}

func lines(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	r, err := db.Query(query)
	require.NoError(t, err)
	defer func() { require.NoError(t, r.Close()) }()
	var out []string
	for r.Next() {
		var line string
		require.NoError(t, r.Scan(&line))
		out = append(out, line)
	}
	require.NoError(t, r.Err())
	return out
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
