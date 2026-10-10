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

func TestListRepos_EachWorkspaceSeesItsOwnInstallations(t *testing.T) {
	repos := []Repo{{Owner: "Acme", FullName: "Acme/api"}, {Owner: "globex", FullName: "globex/web"}}
	store := fakeStore{"acme": {"ws-a"}, "globex": {"ws-b"}}
	tests := []struct {
		name      string
		asApp     bool
		workspace string
		want      []string
	}{
		{"read as the App, a workspace lists its own installation", true, "ws-a", []string{"Acme/api"}},
		{"read as the App, another account's installation lists only in its workspace", true, "ws-b", []string{"globex/web"}},
		{"read as the App, a workspace with no installation lists nothing", true, "ws-c", nil},
		{"with no private key every workspace lists the connected account's view", false, "ws-c", []string{"Acme/api", "globex/web"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(&fakeScanner{repos: repos, asApp: tt.asApp}, store)
			got, err := svc.ListRepos(t.Context(), tt.workspace, "", false)
			require.NoError(t, err)
			var names []string
			for _, r := range got {
				names = append(names, r.FullName)
			}
			assert.Equal(t, tt.want, names)
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

func TestAssignInstallation_IsHeldToConnectorsWriteAndTheWorkspace(t *testing.T) {
	tests := []struct {
		name string
		gate fakeGate
		want error
	}{
		{"a connector manager in the workspace assigns", fakeGate{"": {"ws-a"}, permissions.ConnectorsWrite: {"ws-a"}}, nil},
		{"without connectors:write nothing is assigned", fakeGate{"": {"ws-a"}, permissions.ProjectsWrite: {"ws-a"}}, apperrors.ErrForbidden},
		{"a workspace the caller is not in reads as not found", fakeGate{"": {"ws-b"}, permissions.ConnectorsWrite: {"ws-b"}}, apperrors.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := fakeStore{}
			svc := NewService(Config{Gate: tt.gate, Store: store})
			err := svc.AssignInstallation(t.Context(), "Acme", "ws-a")
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Empty(t, store["acme"])
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{"ws-a"}, store["acme"])

			require.NoError(t, svc.UnassignInstallation(t.Context(), "acme", "ws-a"))
			assert.Empty(t, store["acme"], "unassigning is the way back")
		})
	}
}

func TestClaimInstallation_AssignsOnlyAnInstallationTheInstallerSees(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	newSvc := func(store fakeStore) *Service {
		return NewService(Config{
			Gate:          fakeGate{"": {"ws-a"}, permissions.ProjectsWrite: {"ws-a"}},
			Installations: &fakeScanner{installURL: "https://github.com/apps/nexul-acme/installations/new"},
			Store:         store,
			Installers:    fakeInstallers{42: "globex"},
			StateKey:      []byte("test-key"),
			Now:           func() time.Time { return now },
		})
	}
	link, err := newSvc(fakeStore{}).InstallURL(identity.WithActor(t.Context(), identity.Actor{ID: "alice"}), "ws-a")
	require.NoError(t, err)
	u, err := url.Parse(link)
	require.NoError(t, err)
	state := u.Query().Get("state")
	require.True(t, newSvc(fakeStore{}).ClaimsState(state))

	tests := []struct {
		name           string
		state, code    string
		installationID string
		later          time.Duration
		want           error
	}{
		{"the installer's own installation joins the link's workspace", state, "good-code", "42", 0, nil},
		{"a state naming another workspace is refused", state[:len("install.")] + "ws-b" + state[len("install.ws-a"):], "good-code", "42", 0, apperrors.ErrInvalid},
		{"a link past its week is refused", state, "good-code", "42", 8 * 24 * time.Hour, apperrors.ErrInvalid},
		{"an installation the installer cannot see is refused", state, "good-code", "7", 0, apperrors.ErrForbidden},
		{"no code to confirm the installer is refused", state, "", "42", 0, apperrors.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := fakeStore{}
			svc := newSvc(store)
			svc.cfg.Now = func() time.Time { return now.Add(tt.later) }
			workspaceID, err := svc.ClaimInstallation(t.Context(), tt.state, tt.code, tt.installationID)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				assert.Empty(t, store, "nothing is assigned")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "ws-a", workspaceID)
			assert.Equal(t, fakeStore{"globex": {"ws-a"}}, store)
		})
	}
}
