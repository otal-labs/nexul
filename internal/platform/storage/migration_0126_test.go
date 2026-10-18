package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestMigration0126_ADocsCreatorLocksItOnlyThroughTheirRoles takes docs:lock out of each creator's own grant and
// leaves every other rule on a doc, the creator's deny included, as it was.
func TestMigration0126_ADocsCreatorLocksItOnlyThroughTheirRoles(t *testing.T) {
	db := migrateBefore(t, "0126")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1), ('u-bob', 'bob', 1, 1);
INSERT INTO docs (id, title, body, created_at, updated_at, created_by) VALUES ('d-1', 'Plan', '', 1, 1, 'u-alice');
INSERT INTO permission_overwrites (resource_type, resource_id, target, user_id, allow, deny, created_at, updated_at) VALUES
    ('doc', 'd-1', 'person', 'u-alice', '["docs:delete","docs:lock","docs:read","docs:write","permissions:write"]', '["docs:thread"]', 1, 1),
    ('doc', 'd-1', 'person', 'u-bob', '["docs:lock","docs:read"]', '[]', 1, 1);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))

	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	creator, err := s.Access.Get(t.Context(), "doc", "d-1", "u-alice")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsWrite, permissions.DocsDelete, permissions.PermissionsWrite), creator.Allow)
	assert.Equal(t, permissions.SetOf(permissions.DocsThread), creator.Deny)
	other, err := s.Access.Get(t.Context(), "doc", "d-1", "u-bob")
	require.NoError(t, err)
	assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.DocsLock), other.Allow, "a lock someone was given by name stays")
}
