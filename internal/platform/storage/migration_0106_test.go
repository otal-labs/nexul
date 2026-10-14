package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// TestMigration0106_ExistingRowsGiveNoStepsAndNameNoEnroller upgrades a workspace with its roles, members, runners, an
// open enrollment code and a paired computer: every role keeps its permissions and gives no welcome step, nobody has a
// step record, and no runner or code names who enrolled it.
func TestMigration0106_ExistingRowsGiveNoStepsAndNameNoEnroller(t *testing.T) {
	db := migrateBefore(t, "0106")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('alice', 'alice', 1, 1), ('bob', 'bob', 1, 1);
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1);
INSERT INTO roles (id, workspace_id, name, is_owner_role, created_at, updated_at, permissions) VALUES
    ('role-owner', 'ws-acme', 'Owner', 1, 1, 1, '[]'),
    ('role-dev', 'ws-acme', 'Developer', 0, 2, 2, '["docs:read","tickets:write"]');
INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES ('alice', 'ws-acme', 10), ('bob', 'ws-acme', 11);
INSERT INTO member_roles (user_id, workspace_id, role_id, created_at) VALUES
    ('alice', 'ws-acme', 'role-owner', 10), ('bob', 'ws-acme', 'role-dev', 11);
INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id, owner_user_id, computer_id) VALUES
    ('r-instance', 'instance', 1, 1, 1, 'v1', '', '', ''),
    ('r-deploy', 'm1', 1, 1, 1, 'v1', 'machine-1', '', ''),
    ('r-laptop', 'computer-ab12cd34', 1, 0, 1, 'v1', '', 'bob', 'c-laptop');
INSERT INTO runner_enrollment_codes (code_hash, name, machine, created_at, expires_at, owner_user_id, computer_id) VALUES
    ('hash-1', 'm2', '', 1, 4102444800, '', '');
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at, setup_confirmed_at) VALUES
    ('c-laptop', 'bob', 't3code-v2', 'Laptop', 'http://c-laptop.nexul-computer.invalid', 'sealed', 4102444800, '0.0.46', 5, 5, 6);`)
	require.NoError(t, err)

	migrateUpTo(t, db, "0107") // 0107 gives every role channels:post, which its own test covers.
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))

	devPerms, devSteps := storedRole(t, db, "role-dev")
	assert.Equal(t, permissions.SetOf(permissions.DocsRead, permissions.TicketsWrite), devPerms)
	assert.Empty(t, devSteps)
	_, ownerSteps := storedRole(t, db, "role-owner")
	assert.Empty(t, ownerSteps, "the Owner role's steps live in code, never in its row")

	var records, enrolled int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM welcome_step_states`).Scan(&records))
	assert.Zero(t, records)
	require.NoError(t, db.QueryRow(`SELECT (SELECT COUNT(*) FROM runners WHERE enrolled_by != '') +
		(SELECT COUNT(*) FROM runner_enrollment_codes WHERE enrolled_by != '')`).Scan(&enrolled))
	assert.Zero(t, enrolled)
	laptop, err := s.Runners.GetByComputer(t.Context(), "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, "bob", laptop.OwnerUserID, "a personal runner keeps its person")

	_, err = db.Exec(`INSERT INTO welcome_step_states (user_id, step, skipped_at) VALUES ('bob', 'github', 7)`)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users WHERE id = 'bob'`)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM welcome_step_states`).Scan(&records))
	assert.Zero(t, records, "a person's step records go with them")
}
