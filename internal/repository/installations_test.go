package repository

import (
	"context"
	"net/url"
	"testing"
	"time"

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

func TestListRepos_EachWorkspaceSeesItsOwnInstallationsByAccountID(t *testing.T) {
	repos := []Repo{
		{Owner: "acme-labs", FullName: "acme-labs/api", AccountID: 11},
		{Owner: "acme", FullName: "acme/takeover", AccountID: 99},
		{Owner: "globex", FullName: "globex/web", AccountID: 12},
	}
	live := []Installation{{AccountID: 11, AccountLogin: "acme-labs"}, {AccountID: 99, AccountLogin: "acme"}, {AccountID: 12, AccountLogin: "globex"}}
	tests := []struct {
		name      string
		asApp     bool
		workspace string
		want      []string
	}{
		{"an account renamed on GitHub keeps its workspace, and its old login, taken by another account, does not land there", true, "ws-a", []string{"acme-labs/api"}},
		{"an assignment from before ids were kept is resolved on first read", true, "ws-b", []string{"globex/web"}},
		{"a workspace with no installation lists nothing", true, "ws-c", nil},
		{"with no private key every workspace lists the connected account's view", false, "ws-c", []string{"acme-labs/api", "acme/takeover", "globex/web"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{rows: []Assignment{
				{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a"},
				{AccountLogin: "globex", WorkspaceID: "ws-b"},
			}}
			accounts := &fakeAccounts{installs: live}
			scanner := &fakeScanner{repos: repos, asApp: tt.asApp}
			svc := NewService(Config{Scanner: scanner, Installations: scanner, Accounts: accounts, Store: store})
			got, err := svc.ListRepos(t.Context(), tt.workspace, "", false)
			require.NoError(t, err)
			assert.Equal(t, tt.want, fullNames(got))
			if !tt.asApp {
				return
			}
			_, err = svc.ListRepos(t.Context(), tt.workspace, "", false)
			require.NoError(t, err)
			assert.Equal(t, 1, accounts.lists, "the ids are recorded once, not asked of GitHub on every read")
			assert.Equal(t, int64(12), store.rows[1].AccountID)
		})
	}
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

func TestListInstallations_ShowsEachViewerOnlyWhatTheirWorkspacesHold(t *testing.T) {
	live := []Installation{
		{ID: 1, AccountID: 11, AccountLogin: "acme"},
		{ID: 2, AccountID: 12, AccountLogin: "globex"},
		{ID: 3, AccountID: 13, AccountLogin: "initech"},
	}
	rows := []Assignment{
		{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a", WorkspaceName: "Acme"},
		{AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-a", WorkspaceName: "Acme"},
		{AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-b", WorkspaceName: "Globex"},
	}
	tests := []struct {
		name string
		gate fakeGate
		want map[string][]string
		err  error
	}{
		{
			"a reader in workspace B sees B's installation, named with B alone, and not A's or the unassigned one",
			fakeGate{"": {"ws-b"}, permissions.ConnectorsRead: {"ws-b"}},
			map[string][]string{"globex": {"ws-b"}}, nil,
		},
		{
			"a connector manager also sees the unassigned installation, never one only another workspace holds",
			fakeGate{"": {"ws-b"}, permissions.ConnectorsRead: {"ws-b"}, permissions.ConnectorsWrite: {"ws-b"}},
			map[string][]string{"globex": {"ws-b"}, "initech": nil}, nil,
		},
		{"without connectors:read anywhere the list is refused", fakeGate{"": {"ws-b"}}, nil, apperrors.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := &fakeScanner{installs: live, asApp: true}
			svc := NewService(Config{Gate: tt.gate, Installations: scanner, Accounts: &fakeAccounts{installs: live}, Store: &fakeStore{rows: rows}})
			got, err := svc.ListInstallations(signedIn(t, "bob"))
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			shown := map[string][]string{}
			for _, inst := range got {
				var ids []string
				for _, ws := range inst.Workspaces {
					ids = append(ids, ws.ID)
				}
				shown[inst.AccountLogin] = ids
			}
			assert.Equal(t, tt.want, shown)
		})
	}
}

func TestListInstallations_MarksAnUninstalledAccountGoneAndAReinstallLandsUnassigned(t *testing.T) {
	store := &fakeStore{rows: []Assignment{
		{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a", WorkspaceName: "ws-a"},
		{AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-a", WorkspaceName: "ws-a"},
	}}
	scanner := &fakeScanner{asApp: true, installs: []Installation{{ID: 1, AccountID: 11, AccountLogin: "acme"}},
		repos: []Repo{{FullName: "acme/api", AccountID: 11}, {FullName: "globex/web", AccountID: 12}}}
	svc := NewService(Config{Scanner: scanner, Installations: scanner, Accounts: &fakeAccounts{installs: scanner.installs}, Store: store})

	got, err := svc.ListInstallations(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, Installation{AccountID: 12, AccountLogin: "globex", Gone: true, Workspaces: []InstallationWorkspace{{ID: "ws-a", Name: "ws-a"}}}, got[1],
		"the owner sees the uninstalled account and the workspaces to clear")
	require.Len(t, store.events, 1)
	assert.Equal(t, TopicInstallationUnassigned, store.events[0].Topic)
	assert.Equal(t, InstallationEvent{AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-a", Uninstalled: true}, store.events[0].Payload)
	repos, err := svc.ListRepos(t.Context(), "ws-a", "", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"acme/api"}, fullNames(repos))

	_, err = svc.ListInstallations(t.Context())
	require.NoError(t, err)
	assert.Len(t, store.events, 1, "an account already marked gone publishes nothing more")

	scanner.installs = append(scanner.installs, Installation{ID: 7, AccountID: 12, AccountLogin: "globex"})
	got, err = svc.ListInstallations(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Empty(t, got[1].Workspaces, "a reinstall lands unassigned")
	assert.False(t, got[1].Gone)
}

func TestAssignInstallation_IsHeldToConnectorsWriteWhereTheInstallationIsAndWillBe(t *testing.T) {
	live := []Installation{{AccountID: 11, AccountLogin: "acme"}, {AccountID: 12, AccountLogin: "globex"}}
	tests := []struct {
		name    string
		gate    fakeGate
		account string
		want    error
	}{
		{"a connector manager in the workspace assigns an unassigned installation", fakeGate{"": {"ws-a"}, permissions.ConnectorsWrite: {"ws-a"}}, "Acme", nil},
		{"without connectors:write nothing is assigned", fakeGate{"": {"ws-a"}, permissions.ProjectsWrite: {"ws-a"}}, "acme", apperrors.ErrForbidden},
		{"a workspace the caller is not in reads as not found", fakeGate{"": {"ws-b"}, permissions.ConnectorsWrite: {"ws-b"}}, "acme", apperrors.ErrNotFound},
		{"an installation another workspace holds is not found to someone who cannot manage it there", fakeGate{"": {"ws-a"}, permissions.ConnectorsWrite: {"ws-a"}}, "globex", apperrors.ErrNotFound},
		{"managing it where it is, it is shared", fakeGate{"": {"ws-a", "ws-b"}, permissions.ConnectorsWrite: {"ws-a", "ws-b"}}, "globex", nil},
		{"an account the App is not installed on is not found", fakeGate{"": {"ws-a"}, permissions.ConnectorsWrite: {"ws-a"}}, "initech", apperrors.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{rows: []Assignment{{AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-b"}}}
			svc := NewService(Config{Gate: tt.gate, Accounts: &fakeAccounts{installs: live}, Store: store})
			err := svc.AssignInstallation(signedIn(t, "alice"), tt.account, "ws-a")
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Len(t, store.rows, 1)
				assert.Empty(t, store.events)
				return
			}
			require.NoError(t, err)
			require.Len(t, store.events, 1)
			assert.Equal(t, TopicInstallationAssigned, store.events[0].Topic)
			assert.Equal(t, "ws-a", store.events[0].Payload.(InstallationEvent).WorkspaceID)

			require.NoError(t, svc.UnassignInstallation(signedIn(t, "alice"), tt.account, "ws-a"))
			assert.Len(t, store.rows, 1, "unassigning is the way back")
			require.Len(t, store.events, 2)
			assert.Equal(t, TopicInstallationUnassigned, store.events[1].Topic)
		})
	}
}

func TestClaimInstallation_AssignsOnlyForAnAdminInstallerOnceAndNeverStealsAnAssignedAccount(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	maker := fakeGate{"": {"ws-a", "ws-b"}, permissions.ProjectsWrite: {"ws-a"}}
	installers := fakeInstallers{
		42: {AccountID: 12, AccountLogin: "Globex", Admin: true},
		43: {AccountID: 13, AccountLogin: "initech", Admin: false},
		44: {AccountID: 11, AccountLogin: "acme", Admin: true},
	}
	newSvc := func(store *fakeStore, gate fakeGate) *Service {
		return NewService(Config{
			Gate:          gate,
			Installations: &fakeScanner{installURL: "https://github.com/apps/nexul-acme/installations/new"},
			Store:         store, States: store,
			Installers: installers,
			Now:        func() time.Time { return now },
		})
	}
	mint := func(t *testing.T, store *fakeStore) string {
		link, err := newSvc(store, maker).InstallURL(signedIn(t, "alice"), "ws-a")
		require.NoError(t, err)
		u, err := url.Parse(link)
		require.NoError(t, err)
		return u.Query().Get("state")
	}

	tests := []struct {
		name           string
		claimGate      fakeGate
		installationID string
		later          time.Duration
		replay         bool
		want           error
	}{
		{"the installer administering the account assigns it to the link's workspace", maker, "42", 0, false, nil},
		{"a link works once", maker, "42", 0, true, apperrors.ErrInvalid},
		{"an installer who is not shown to administer the account leaves it unassigned", maker, "43", 0, false, apperrors.ErrForbidden},
		{"a link whose maker can no longer add projects there assigns nothing", fakeGate{"": {"ws-a"}}, "42", 0, false, apperrors.ErrForbidden},
		{"an account another workspace holds is not taken", maker, "44", 0, false, apperrors.ErrConflict},
		{"a link past its day is refused", maker, "42", 25 * time.Hour, false, apperrors.ErrInvalid},
		{"an installation the installer cannot see is refused", maker, "7", 0, false, apperrors.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{rows: []Assignment{{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-b"}}}
			state := mint(t, store)
			svc := newSvc(store, tt.claimGate)
			svc.cfg.Now = func() time.Time { return now.Add(tt.later) }
			if tt.replay {
				_, err := svc.ClaimInstallation(t.Context(), state, "good-code", tt.installationID)
				require.NoError(t, err)
			}
			workspaceID, err := svc.ClaimInstallation(t.Context(), state, "good-code", tt.installationID)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				if !tt.replay {
					assert.Equal(t, []Assignment{{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-b"}}, store.rows, "nothing is assigned")
				}
				_, err = svc.ClaimInstallation(t.Context(), state, "good-code", tt.installationID)
				require.ErrorIs(t, err, apperrors.ErrInvalid, "a refused claim still uses the link up")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "ws-a", workspaceID)
			assert.Contains(t, store.rows, Assignment{AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-a", WorkspaceName: "ws-a"})
			require.Len(t, store.events, 1)
			assert.Equal(t, "alice", store.events[0].Payload.(InstallationEvent).ActorID, "the link's maker is the actor")
		})
	}

	t.Run("a state this instance never made is refused", func(t *testing.T) {
		store := &fakeStore{}
		_, err := newSvc(store, maker).ClaimInstallation(t.Context(), "install.forged", "good-code", "42")
		require.ErrorIs(t, err, apperrors.ErrInvalid)
		assert.Empty(t, store.rows)
	})
}

func TestScan_RejectsAnInstallationOutsideTheCallersWorkspaces(t *testing.T) {
	for _, tt := range []struct {
		name  string
		asApp bool
		owner string
		want  error
	}{
		{"another workspace's installation is hidden", true, "globex", apperrors.ErrNotFound},
		{"an unassigned installation is hidden", true, "initech", apperrors.ErrNotFound},
		{"an assigned installation is readable ignoring account case", true, "ACME", nil},
		{"without a key the connector's existing access survives", false, "globex", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			scanner := &fakeScanner{asApp: tt.asApp, resolvedRef: "main"}
			accounts := &fakeAccounts{installs: []Installation{{AccountID: 11, AccountLogin: "acme"}, {AccountID: 12, AccountLogin: "globex"}, {AccountID: 13, AccountLogin: "initech"}}}
			svc := NewService(Config{
				Gate:    fakeGate{"": {"ws-a", "ws-b"}, permissions.ProjectsWrite: {"ws-a", "ws-b"}},
				Scanner: scanner, Installations: scanner, Accounts: accounts,
				Store: &fakeStore{rows: []Assignment{{AccountID: 11, AccountLogin: "acme", WorkspaceID: "ws-a"}, {AccountID: 12, AccountLogin: "globex", WorkspaceID: "ws-b"}}},
			})
			got, err := svc.Scan(t.Context(), "ws-a", tt.owner, "api", "")
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "main", got.DefaultBranch)
		})
	}
}
