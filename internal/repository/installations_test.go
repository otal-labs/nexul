package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func fullNames(repos []Repo) []string {
	var out []string
	for _, r := range repos {
		out = append(out, r.FullName)
	}
	return out
}

// fakeGate holds each action in the workspaces listed for it; "" is membership.
type fakeGate map[permissions.Action][]string

func (g fakeGate) holds(action permissions.Action, workspaceID string) bool {
	for _, id := range g[action] {
		if id == workspaceID {
			return true
		}
	}
	return false
}

func (g fakeGate) RequireAnywhere(_ context.Context, action permissions.Action) error {
	if len(g[action]) == 0 {
		return apperrors.ErrForbidden
	}
	return nil
}

func (g fakeGate) Require(_ context.Context, workspaceID string, action permissions.Action) error {
	if !g.holds("", workspaceID) {
		return apperrors.ErrNotFound
	}
	if action != "" && !g.holds(action, workspaceID) {
		return apperrors.ErrForbidden
	}
	return nil
}

func (g fakeGate) WorkspacesWith(_ context.Context, action permissions.Action) ([]string, error) {
	return g[action], nil
}

func signedIn(t *testing.T, userID string) context.Context {
	return identity.WithActor(t.Context(), identity.Actor{ID: userID})
}

// aliceAndBob is an instance reading GitHub as its App, where Bob's account is linked to Alice's workspace: Alice's
// own GitHub view holds only acme/api, and Bob's holds his private bob/secret.
func aliceAndBob() (*fakePeople, *fakeStore, *fakeScanner) {
	alice := &fakeScanner{asApp: true, repos: []Repo{{Owner: "acme", FullName: "acme/api", AccountID: 11}},
		installs: []Installation{{ID: 1, AccountID: 11, AccountLogin: "acme"}}}
	bob := &fakeScanner{repos: []Repo{{Owner: "bob", FullName: "bob/secret", AccountID: 22}},
		installs: []Installation{{ID: 2, AccountID: 22, AccountLogin: "bob"}}}
	store := &fakeStore{
		rows: []Assignment{
			{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a", WorkspaceName: "Acme"},
			{AccountID: 22, AccountLogin: "bob", WorkspaceID: "ws-a", WorkspaceName: "Acme"},
		},
		attached: map[string][]string{"acme": {"ws-a"}, "bob": {"ws-a"}},
	}
	return &fakePeople{views: map[string]*fakeScanner{"alice": alice, "bob": bob}}, store, alice
}

var aliceWrites = fakeGate{"": {"ws-a"}, permissions.ProjectsRead: {"ws-a"}, permissions.ProjectsWrite: {"ws-a"}}

func TestListRepos_ListsOnlyTheCallersOwnGitHubViewWhateverIsLinkedToTheirWorkspace(t *testing.T) {
	people, store, alice := aliceAndBob()
	svc := NewService(Config{Gate: aliceWrites, People: people, Installations: alice, Accounts: &fakeAccounts{}, Store: store})

	got, err := svc.ListRepos(signedIn(t, "alice"), "ws-a", "", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"acme/api"}, fullNames(got), "Bob's private repository never lists for Alice, though his account is linked to her workspace")

	_, err = svc.ListRepos(signedIn(t, "carol"), "ws-a", "", false)
	require.ErrorIs(t, err, errNotConnected, "someone who has not connected GitHub gets no list at all")
	assert.Equal(t, []string{"alice", "carol"}, people.asked, "each list reads only the caller's own view, never another person's")
}

func TestScan_ReadsWithTheCallersOwnGitHubView(t *testing.T) {
	people, store, alice := aliceAndBob()
	alice.resolvedRef = "main"
	svc := NewService(Config{Gate: aliceWrites, People: people, Installations: alice, Accounts: &fakeAccounts{}, Store: store})

	got, err := svc.Scan(signedIn(t, "alice"), "ws-a", "acme", "api", "")
	require.NoError(t, err)
	assert.Equal(t, "main", got.DefaultBranch)

	_, err = svc.Scan(signedIn(t, "carol"), "ws-a", "bob", "secret", "")
	require.ErrorIs(t, err, errNotConnected)
}

func TestRequireAttach_LinksTheAccountOnlyForARepositoryTheAttacherCanOpen(t *testing.T) {
	tests := []struct {
		name        string
		owner, repo string
		want        error
	}{
		{"a repository the attacher's own token lists links its account, ignoring case", "ACME", "API", nil},
		{"another person's repository is refused, even with their account linked already", "bob", "secret", apperrors.ErrNotFound},
		{"an attacher who has not connected GitHub is refused", "acme", "api", errNotConnected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			people, _, alice := aliceAndBob()
			store := &fakeStore{}
			svc := NewService(Config{Gate: aliceWrites, People: people, Installations: alice, Accounts: &fakeAccounts{}, Store: store})
			caller := "alice"
			if tt.want == errNotConnected {
				caller = "carol"
			}
			err := svc.RequireAttach(signedIn(t, caller), "ws-a", tt.owner, tt.repo)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Empty(t, store.rows, "nothing is linked")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []Assignment{{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a", WorkspaceName: "ws-a"}}, store.rows)
			require.Len(t, store.events, 1)
			assert.Equal(t, InstallationEvent{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a", ActorID: "alice"}, store.events[0].Payload)
		})
	}

	t.Run("a miss asks GitHub again once before refusing, for an App just installed", func(t *testing.T) {
		people, _, alice := aliceAndBob()
		svc := NewService(Config{People: people, Installations: alice, Accounts: &fakeAccounts{}, Store: &fakeStore{}})
		require.ErrorIs(t, svc.RequireAttach(signedIn(t, "alice"), "ws-a", "acme", "ghost"), apperrors.ErrNotFound)
		assert.Equal(t, 2, alice.listCalls)
		assert.True(t, alice.lastRefresh)
	})
}

func TestRequireAssigned_HoldsBackgroundWorkToTheAccountsLinkedToTheWorkspace(t *testing.T) {
	live := []Installation{{AccountID: 11, AccountLogin: "acme-labs"}, {AccountID: 99, AccountLogin: "acme"}, {AccountID: 12, AccountLogin: "globex"}}
	tests := []struct {
		name      string
		asApp     bool
		workspace string
		owner     string
		want      error
	}{
		{"an account renamed on GitHub keeps its workspace", true, "ws-a", "acme-labs", nil},
		{"its old login, taken by another account, does not land there", true, "ws-a", "acme", apperrors.ErrNotFound},
		{"a link from before ids were kept is resolved on first read", true, "ws-b", "globex", nil},
		{"a workspace with no link reads nothing", true, "ws-c", "globex", apperrors.ErrNotFound},
		{"with no private key the connector's existing access survives", false, "ws-c", "globex", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{rows: []Assignment{
				{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a"},
				{AccountLogin: "globex", WorkspaceID: "ws-b"},
			}}
			accounts := &fakeAccounts{installs: live}
			svc := NewService(Config{Installations: &fakeScanner{asApp: tt.asApp}, Accounts: accounts, Store: store})
			err := svc.RequireAssigned(t.Context(), tt.workspace, tt.owner, "api")
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestListInstallations_ShowsOnlyTheViewersOwnInstallationsAndTheWorkspacesThatUseThem(t *testing.T) {
	people, store, alice := aliceAndBob()
	store.rows = append(store.rows,
		Assignment{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-b", WorkspaceName: "Globex"},
		Assignment{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-c", WorkspaceName: "Initech"},
	)
	store.attached["acme"] = []string{"ws-a", "ws-b"}
	gate := fakeGate{"": {"ws-a", "ws-b", "ws-c"}, permissions.ProjectsRead: {"ws-a", "ws-b", "ws-c"}, permissions.ProjectsWrite: {"ws-a"}}
	svc := NewService(Config{Gate: gate, People: people, Installations: alice, Accounts: &fakeAccounts{installs: alice.installs}, Store: store})

	got, err := svc.ListInstallations(signedIn(t, "alice"))
	require.NoError(t, err)
	require.Len(t, got, 1, "Bob's account never appears to Alice")
	assert.Equal(t, "acme", got[0].AccountLogin)
	assert.Equal(t, []InstallationWorkspace{{ID: "ws-a", Name: "Acme", CanDetach: true}, {ID: "ws-b", Name: "Globex"}}, got[0].Workspaces,
		"a workspace whose projects attach none of its repositories does not use it, and detaching takes projects:write")

	_, err = svc.ListInstallations(signedIn(t, "carol"))
	require.ErrorIs(t, err, errNotConnected)
}

func TestListInstallations_MarksAnUninstalledAccountGoneAndAReinstallLinksNothing(t *testing.T) {
	people, store, alice := aliceAndBob()
	accounts := &fakeAccounts{installs: []Installation{{ID: 1, AccountID: 11, AccountLogin: "acme"}}}
	svc := NewService(Config{Gate: aliceWrites, People: people, Installations: alice, Accounts: accounts, Store: store})

	_, err := svc.ListInstallations(signedIn(t, "alice"))
	require.NoError(t, err)
	require.Len(t, store.events, 1)
	assert.Equal(t, TopicInstallationUnassigned, store.events[0].Topic)
	assert.Equal(t, InstallationEvent{AccountID: 22, AccountLogin: "bob", WorkspaceID: "ws-a", Uninstalled: true}, store.events[0].Payload)
	require.ErrorIs(t, svc.RequireAssigned(t.Context(), "ws-a", "bob", "secret"), apperrors.ErrNotFound, "background work stops on a gone account")

	_, err = svc.ListInstallations(signedIn(t, "alice"))
	require.NoError(t, err)
	assert.Len(t, store.events, 1, "an account already marked gone publishes nothing more")

	accounts.installs = append(accounts.installs, Installation{ID: 7, AccountID: 22, AccountLogin: "bob"})
	_, err = svc.ListInstallations(signedIn(t, "alice"))
	require.NoError(t, err)
	assert.NotContains(t, store.rows, Assignment{AccountID: 22, AccountLogin: "bob", WorkspaceID: "ws-a", WorkspaceName: "Acme", Gone: true}, "a reinstall drops the gone link instead of reviving it")
}

func TestUnassignInstallation_DetachesWithProjectsWriteInThatWorkspace(t *testing.T) {
	tests := []struct {
		name string
		gate fakeGate
		want error
	}{
		{"projects:write there detaches the account", aliceWrites, nil},
		{"without projects:write nothing is detached", fakeGate{"": {"ws-a"}, permissions.ConnectorsWrite: {"ws-a"}}, apperrors.ErrForbidden},
		{"a workspace the caller is not in reads as not found", fakeGate{"": {"ws-b"}, permissions.ProjectsWrite: {"ws-b"}}, apperrors.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{rows: []Assignment{{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a"}}}
			svc := NewService(Config{Gate: tt.gate, Store: store})
			err := svc.UnassignInstallation(signedIn(t, "alice"), "Acme", "ws-a")
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Len(t, store.rows, 1)
				return
			}
			require.NoError(t, err)
			assert.Empty(t, store.rows)
			require.Len(t, store.events, 1)
			assert.Equal(t, TopicInstallationUnassigned, store.events[0].Topic)
		})
	}
}
