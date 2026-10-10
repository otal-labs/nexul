package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/repository"
)

// TestMigration0090_AssignmentsKeepTheirWorkspacesUntilTheFirstReadRecordsTheirIDs upgrades a database at the previous
// schema: every login-keyed assignment survives without an id, and recording the ids keeps one row per workspace.
func TestMigration0090_AssignmentsKeepTheirWorkspacesUntilTheFirstReadRecordsTheirIDs(t *testing.T) {
	db := migrateBefore(t, "0090")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1), ('ws-globex', 'Globex', 'globex', 1, 1);
INSERT INTO github_installation_workspaces (account_login, workspace_id, assigned_at) VALUES
    ('acme', 'ws-acme', 1), ('acme', 'ws-globex', 1), ('globex', 'ws-globex', 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0090 and every later migration apply on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()
	rows, err := s.GitHubInstallations.ListAssignments(ctx)
	require.NoError(t, err)
	assert.Equal(t, []repository.Assignment{
		{AccountLogin: "acme", WorkspaceID: "ws-acme", WorkspaceName: "Acme"},
		{AccountLogin: "acme", WorkspaceID: "ws-globex", WorkspaceName: "Globex"},
		{AccountLogin: "globex", WorkspaceID: "ws-globex", WorkspaceName: "Globex"},
	}, rows)
	pending, err := s.GitHubInstallations.HasUnresolvedAssignments(ctx)
	require.NoError(t, err)
	assert.True(t, pending, "the first read as the App resolves them")

	changed, err := s.GitHubInstallations.AssignInstallation(ctx, repository.Assignment{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-acme"})
	require.NoError(t, err)
	assert.True(t, changed)
	require.NoError(t, s.GitHubInstallations.SyncAccounts(ctx, repository.AccountSync{
		Resolved: map[string]int64{"acme": 11},
		Gone:     []repository.Assignment{{AccountLogin: "globex", WorkspaceID: "ws-globex"}},
	}))
	rows, err = s.GitHubInstallations.ListAssignments(ctx)
	require.NoError(t, err)
	assert.Equal(t, []repository.Assignment{
		{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-acme", WorkspaceName: "Acme"},
		{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-globex", WorkspaceName: "Globex"},
		{AccountLogin: "globex", WorkspaceID: "ws-globex", WorkspaceName: "Globex", Gone: true},
	}, rows, "one row per workspace once the id is known, and an unresolvable login is gone")
	ids, err := s.GitHubInstallations.AssignedAccountsIn(ctx, []string{"ws-globex"})
	require.NoError(t, err)
	assert.Equal(t, []int64{11}, ids)
}

func TestGitHubInstallationsRepo_WritesItsEventOnlyWhenTheAssignmentChanges(t *testing.T) {
	s := openConnectorAppConfigStore(t)
	ctx := t.Context()
	event := func() eventbus.OutboxEvent {
		return eventbus.OutboxEvent{ID: time.Now().Format(time.RFC3339Nano), Topic: repository.TopicInstallationAssigned, Payload: repository.InstallationEvent{AccountID: 11}}
	}
	outbox := func() int {
		var n int
		require.NoError(t, s.db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE topic LIKE 'repository.installation.%'`).Scan(&n))
		return n
	}
	a := repository.Assignment{AccountID: 11, AccountLogin: "Acme", WorkspaceID: "workspace-default"}

	changed, err := s.GitHubInstallations.AssignInstallation(ctx, a, event())
	require.NoError(t, err)
	assert.True(t, changed)
	changed, err = s.GitHubInstallations.AssignInstallation(ctx, a, event())
	require.NoError(t, err)
	assert.False(t, changed, "assigning twice is a no-op")
	assert.Equal(t, 1, outbox())

	require.NoError(t, s.GitHubInstallations.SyncAccounts(ctx, repository.AccountSync{Gone: []repository.Assignment{a}}))
	changed, err = s.GitHubInstallations.AssignInstallation(ctx, a, event())
	require.NoError(t, err)
	assert.True(t, changed, "assigning revives a gone assignment")
	assert.Equal(t, 2, outbox())

	changed, err = s.GitHubInstallations.UnassignInstallation(ctx, a, event())
	require.NoError(t, err)
	assert.True(t, changed)
	changed, err = s.GitHubInstallations.UnassignInstallation(ctx, a, event())
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, 3, outbox())
}
