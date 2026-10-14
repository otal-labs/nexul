package storage

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// migrateBefore applies every embedded migration older than version, leaving the database as a binary from
// before version would have.
func migrateBefore(t *testing.T, version string) *sql.DB {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, ensureMigrationsTable(db))
	versions, err := embeddedVersions()
	require.NoError(t, err)
	for _, v := range versions {
		if v >= version {
			break
		}
		script, err := migrationFS.ReadFile("migrations/" + v + ".sql")
		require.NoError(t, err)
		require.NoError(t, applyMigration(db, v, string(script)))
	}
	return db
}

// migrateUpTo applies the pending embedded migrations older than version, so a test of an earlier step reads its rows
// before a later migration rewrites them.
func migrateUpTo(t *testing.T, db *sql.DB, version string) {
	t.Helper()
	pending, err := Pending(db)
	require.NoError(t, err)
	for _, v := range pending {
		if v >= version {
			return
		}
		script, err := migrationFS.ReadFile("migrations/" + v + ".sql")
		require.NoError(t, err)
		require.NoError(t, applyMigration(db, v, string(script)))
	}
}

// withArchivedAt adds workspaces.archived_at to a database a test stopped short of, for repo reads that select it.
func withArchivedAt(t *testing.T, db *sql.DB) {
	t.Helper()
	script, err := migrationFS.ReadFile("migrations/0113_workspace_archived_at.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(script))
	require.NoError(t, err)
}

// withRoleDescription adds roles.description to a database a test stopped short of, for repo reads that select it.
func withRoleDescription(t *testing.T, db *sql.DB) {
	t.Helper()
	script, err := migrationFS.ReadFile("migrations/0116_role_descriptions.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(script))
	require.NoError(t, err)
}

// storedRole reads a role's permissions and welcome steps straight from its row, for a test that stops before later
// migrations that add columns the Roles repo already selects.
func storedRole(t *testing.T, db *sql.DB, id string) (permissions.Set, []string) {
	t.Helper()
	var rawPerms, rawSteps string
	require.NoError(t, db.QueryRow(`SELECT permissions, welcome_steps FROM roles WHERE id = ?`, id).Scan(&rawPerms, &rawSteps))
	perms, err := parseSet(rawPerms)
	require.NoError(t, err)
	var steps []string
	require.NoError(t, json.Unmarshal([]byte(rawSteps), &steps))
	return perms, steps
}

// storedAutomationScopes reads an automation's scopes from its row, since the repo reads columns a migration under test may predate.
func storedAutomationScopes(t *testing.T, db *sql.DB, id string) []string {
	t.Helper()
	var raw []byte
	require.NoError(t, db.QueryRow(`SELECT scopes FROM automations WHERE id = ?`, id).Scan(&raw))
	scopes, err := decodeStringList(raw)
	require.NoError(t, err)
	return scopes
}

func apply0029(t *testing.T, db *sql.DB) {
	t.Helper()
	script, err := migrationFS.ReadFile("migrations/0029_instance_stacks.sql")
	require.NoError(t, err)
	require.NoError(t, applyMigration(db, "0029_instance_stacks", string(script)))
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(query, args...).Scan(&n))
	return n
}

func TestApplyMigration_DanglingReference_RollsBack(t *testing.T) {
	db := newTestDB(t)
	db.SetMaxOpenConns(1)

	err := applyMigration(db, "9999_dangling",
		`INSERT INTO statuses (id, project_id, name, position, kind, created_at, updated_at) VALUES ('s-ghost', 'ghost', 'Ghost', 0, 'backlog', 0, 0);`)

	require.ErrorContains(t, err, "references a missing projects")
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM statuses WHERE id = 's-ghost'`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM schema_migrations WHERE version = '9999_dangling'`))
	_, err = db.Exec(`INSERT INTO statuses (id, project_id, name, position, kind, created_at, updated_at) VALUES ('s-ghost', 'ghost', 'Ghost', 0, 'backlog', 0, 0)`)
	require.Error(t, err, "foreign keys are enforced again on the connection the migration borrowed")
}

func TestMigrate_FreshInstance_SeedsAWorkspaceButNoProject(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, Migrate(db))

	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM workspaces WHERE id = 'workspace-default'`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM projects`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM statuses`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM ticket_types`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM memories`))
}

func TestMigration0029_MovesGatewayStacksToTheInstance(t *testing.T) {
	db := migrateBefore(t, "0029")
	_, err := db.Exec(`
INSERT INTO users (id, provider, provider_user_id, login, created_at, updated_at) VALUES ('u1', 'github', '1', 'owner', 0, 0);
INSERT INTO stacks (id, project_id, name, slug, machine, strategy, created_at, updated_at) VALUES
    ('st-tunnel', 'project-general', 'cloudflared-instance', 'cloudflared-instance', 'box', 'run', 0, 0),
    ('st-proxy', 'project-general', 'nexul-proxy', 'nexul-proxy', 'box', 'run', 0, 0),
    ('st-app', 'project-general', 'app', 'app', 'box', 'compose', 0, 0);
INSERT INTO services (id, stack_id, name) VALUES ('c-tunnel', 'st-tunnel', 'cloudflared'), ('c-app', 'st-app', 'web');
INSERT INTO dns_tunnels (id, name, token, agent_service_id, created_at, updated_at) VALUES ('t1', 'instance', '', 'st-tunnel', 0, 0);
INSERT INTO dns_gateways (id, kind, docker_network, service_id, created_at, updated_at) VALUES ('g1', 'proxy', 'nexul_proxy', 'st-proxy', 0, 0);
`)
	require.NoError(t, err)

	apply0029(t, db)

	assert.Equal(t, 2, count(t, db, `SELECT COUNT(*) FROM stacks WHERE project_id IS NULL AND id IN ('st-tunnel', 'st-proxy')`))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM stacks WHERE id = 'st-app' AND project_id = 'project-general'`))
	assert.Equal(t, 2, count(t, db, `SELECT COUNT(*) FROM services`), "rebuilding the table never cascades to its containers")
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM projects WHERE id = 'project-general'`),
		"an instance someone signed in to keeps the project they were given")
	_, err = db.Exec(`INSERT INTO stacks (id, project_id, name, slug, machine, strategy, created_at, updated_at)
		VALUES ('st-bad', 'ghost', 'bad', 'bad', 'box', 'run', 0, 0)`)
	require.Error(t, err, "a stack's project is still a foreign key when set")
}

func TestMigration0029_UnclaimedInstance_LosesTheSeededProject(t *testing.T) {
	db := migrateBefore(t, "0029")
	require.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM projects WHERE id = 'project-general'`))

	apply0029(t, db)

	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM projects`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM statuses`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM ticket_types`))
	assert.Zero(t, count(t, db, `SELECT COUNT(*) FROM memories`))
}
