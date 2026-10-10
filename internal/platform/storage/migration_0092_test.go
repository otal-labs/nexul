package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/repository"
)

// TestMigration0092_KeepsTheLinksAndDropsTheInstallLinkStates upgrades a database at the previous schema: an install
// link's state goes, every link stays, and a person's GitHub token is stored encrypted and dropped with its person.
func TestMigration0092_KeepsTheLinksAndDropsTheInstallLinkStates(t *testing.T) {
	db := migrateBefore(t, "0092")
	_, err := db.Exec(`
INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-acme', 'Acme', 'acme', 1, 1), ('ws-globex', 'Globex', 'globex', 1, 1);
INSERT INTO projects (id, name, prefix, position, workspace_id, created_at, updated_at) VALUES ('p-api', 'API', 'API', 1, 'ws-acme', 1, 1);
INSERT INTO project_repos (project_id, owner, name, full_name, added_at, connector_id) VALUES ('p-api', 'Acme', 'api', 'Acme/api', 1, 'github');
INSERT INTO github_installation_workspaces (account_id, account_login, workspace_id, assigned_at) VALUES (11, 'acme', 'ws-acme', 1), (11, 'acme', 'ws-globex', 1);
INSERT INTO github_install_states (state_hash, workspace_id, user_id, expires_at) VALUES ('hash-1', 'ws-acme', 'alice', 1);
INSERT INTO users (id, login, created_at, updated_at) VALUES ('alice', 'alice', 1, 1);
`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0092 and every later migration apply on top, as an upgrade would")
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'github_install_states'`).Scan(&n))
	assert.Zero(t, n)

	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()
	rows, err := s.GitHubInstallations.AssignmentsIn(ctx, []string{"ws-acme", "ws-globex"})
	require.NoError(t, err)
	assert.Equal(t, []repository.Assignment{
		{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-acme", WorkspaceName: "Acme", Attached: true},
		{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-globex", WorkspaceName: "Globex"},
	}, rows, "a workspace uses an account while one of its projects attaches a repository of it, whatever the case")

	_, err = s.GitHubUserLinks.GetGitHubLink(ctx, "alice")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "nobody has a token until they sign in with GitHub or connect it")
	link := auth.GitHubLink{UserID: "alice", AccessToken: "ghu_secret", RefreshToken: "ghr_secret",
		ExpiresAt: time.Unix(1_800_000_000, 0).UTC(), ConnectedAt: time.Unix(1_700_000_000, 0).UTC()}
	require.NoError(t, s.GitHubUserLinks.SaveGitHubLink(ctx, link))
	got, err := s.GitHubUserLinks.GetGitHubLink(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, link, got)
	var stored string
	require.NoError(t, db.QueryRow(`SELECT access_token || refresh_token FROM github_user_links WHERE user_id = 'alice'`).Scan(&stored))
	assert.NotContains(t, stored, "secret", "tokens are encrypted at rest")

	_, err = db.Exec(`DELETE FROM users WHERE id = 'alice'`)
	require.NoError(t, err)
	_, err = s.GitHubUserLinks.GetGitHubLink(ctx, "alice")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a removed person's token goes with them")
}
