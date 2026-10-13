package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestMigration0097_EveryRuleSurvivesAsAPersonsRule upgrades a database at the previous schema holding every kind of
// row the table has carried: each comes through unchanged, names its person, and reads back as it did.
func TestMigration0097_EveryRuleSurvivesAsAPersonsRule(t *testing.T) {
	db := migrateBefore(t, "0097")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('alice', 'alice', 1, 1), ('bob', 'bob', 1, 1), ('sam', 'sam', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-store', 'Storefront', 'STO', 1, 'workspace-default', 1, 1);
INSERT INTO permission_overwrites (resource_type, resource_id, user_id, allow, deny, created_at, updated_at) VALUES
    ('doc', 'd-price', 'alice', ?, '[]', 10, 11),
    ('doc', 'd-price', 'bob', '["docs:read"]', '[]', 12, 13),
    ('doc', 'd-plan', 'bob', '[]', '["docs:write"]', 14, 15),
    ('play', 'play-notes', 'sam', '[]', '["plays:run"]', 16, 17),
    ('project', 'p-store', 'sam', '["tickets:read","tickets:write"]', '[]', 18, 19),
    ('workspace', 'workspace-default', 'bob', '["tickets:write"]', '["docs:delete"]', 20, 21);
`, setJSON(permissions.CreatorGrant))
	require.NoError(t, err)
	rows := func(query string) []string {
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
	const columns = `resource_type || '|' || resource_id || '|' || user_id || '|' || allow || '|' || deny || '|' || created_at || '|' || updated_at`
	before := rows(`SELECT ` + columns + ` FROM permission_overwrites ORDER BY 1`)

	migrateUpTo(t, db, "0100")

	assert.Equal(t, before, rows(`SELECT `+columns+` FROM permission_overwrites ORDER BY 1`))
	assert.Equal(t, []string{"person|6"}, rows(`SELECT target || '|' || COUNT(*) FROM permission_overwrites WHERE role_id IS NULL GROUP BY target`))

	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()
	docs, err := s.Access.Rules(ctx, "doc", []string{"d-price", "d-plan"}, "bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead), docs["d-price"].Person.Allow, "a doc share")
	assert.Equal(t, permissions.SetOf(permissions.DocsWrite), docs["d-plan"].Person.Deny, "a doc's deny")
	creator, err := s.Access.Get(ctx, "doc", "d-price", "alice")
	require.NoError(t, err)
	assert.Equal(t, permissions.CreatorGrant, creator.Allow, "the creator's own grant")
	plays, err := s.Access.RulesOfType(ctx, "play", "sam")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.PlaysRun), plays["play-notes"].Person.Deny, "a play exclusion")
	projectAccess, err := s.Access.Get(ctx, "project", "p-store", "sam")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.TicketsRead, permissions.TicketsWrite), projectAccess.Allow, "Project access")
	override, err := s.Access.Get(ctx, "workspace", "workspace-default", "bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.TicketsWrite), override.Allow, "a workspace-wide allow")
	assert.Equal(t, permissions.SetOf(permissions.DocsDelete), override.Deny, "a workspace-wide deny")

	require.NoError(t, s.Access.Set(ctx, "doc", "d-price", "bob", permissions.SetOf(permissions.DocsWrite), nil))
	assert.Equal(t, []string{"bob|1"}, rows(`SELECT user_id || '|' || COUNT(*) FROM permission_overwrites WHERE resource_id = 'd-price' AND user_id = 'bob' GROUP BY user_id`),
		"writing a kept row updates it in place")
}
