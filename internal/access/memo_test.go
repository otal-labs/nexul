package access

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// flakyRoles fails every membership lookup while failing is set, the way a busy database would.
type flakyRoles struct {
	*fakeRoles
	failing bool
}

func (f *flakyRoles) MemberRole(ctx context.Context, workspaceID, userID string) (RoleInfo, error) {
	if f.failing {
		return RoleInfo{}, errors.New("database is locked")
	}
	return f.fakeRoles.MemberRole(ctx, workspaceID, userID)
}

func TestMemo_AFailedReadIsNotRemembered(t *testing.T) {
	s, repo := restrictedFixture(t)
	roles := &flakyRoles{fakeRoles: newFakeRoles(), failing: true}
	roles.set("ws", "team", RoleInfo{Permissions: permissions.SetOf(permissions.TicketsRead)})
	s.SetRoles(roles)
	repo.getErr = errors.New("database is locked")
	ctx := WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: "team"}), s.NewMemo())

	require.Error(t, s.RequireProject(ctx, "p-open", permissions.TicketsRead))
	roles.failing, repo.getErr = false, nil

	assert.NoError(t, s.RequireProject(ctx, "p-open", permissions.TicketsRead), "the same request asks again once the read works")
}

func TestProjectsWith_AnswersAsEachProjectsCheck(t *testing.T) {
	s, repo := restrictedFixture(t)
	require.NoError(t, repo.Set(t.Context(), "workspace", "ws", "team", nil, permissions.SetOf(permissions.TicketsWrite)))
	actions := []permissions.Action{permissions.Member, permissions.TicketsRead, permissions.TicketsWrite, permissions.RunnersRead, permissions.ChatWrite}

	for _, user := range []string{"client", "team", "owner", "stranger"} {
		for _, action := range actions {
			got, err := s.ProjectsWith(t.Context(), user, "ws", action)
			require.NoError(t, err)
			var want []string
			for _, projectID := range []string{"p-hidden", "p-open"} {
				if s.CanInProject(t.Context(), user, projectID, action) {
					want = append(want, projectID)
				}
			}
			assert.ElementsMatch(t, want, got, "%s %q", user, action)
		}
	}
}

func TestProjectsWith_ReadsARestrictedMembersAccessOnce(t *testing.T) {
	s, repo := restrictedFixture(t)
	repo.resetReads()

	got, err := s.ProjectsWith(t.Context(), "client", "ws", permissions.TicketsRead)

	require.NoError(t, err)
	assert.Equal(t, []string{"p-open"}, got)
	assert.Equal(t, readCounts{get: 1, getMany: 1}, repo.readCounts(), "the workspace overwrite, then every project's access at once")
}

// TestCallerProjects_AnswersAsRequireProject: the set a list filters by in SQL is exactly the projects RequireProject
// lets each kind of caller through, so a list filtered by it shows what the row-by-row check showed (ADR 0140).
func TestCallerProjects_AnswersAsRequireProject(t *testing.T) {
	s, repo := restrictedFixture(t)
	require.NoError(t, repo.Set(t.Context(), "workspace", "ws", "team", nil, permissions.SetOf(permissions.TicketsWrite)))
	s.SetScopes(fakeScopes{
		projects:   map[string]string{"p-open": "ws", "p-hidden": "ws", "p-other": "ws-2"},
		workspaces: map[string][]string{"team": {"ws"}},
		restricted: map[string][]string{"client": {"ws"}, "owner": {"ws"}},
	})
	callers := map[string]context.Context{
		"no actor":           context.Background(),
		"default automation": identity.WithActor(context.Background(), identity.Actor{Automation: &identity.AutomationRef{ID: "a-1", WorkspaceID: "ws"}}),
		"anonymous actor":    identity.WithActor(context.Background(), identity.Actor{}),
	}
	for _, user := range []string{"client", "team", "owner", "stranger"} {
		callers[user] = identity.WithActor(context.Background(), identity.Actor{ID: user})
	}
	actions := []permissions.Action{permissions.Member, permissions.TicketsRead, permissions.TicketsWrite, permissions.DocsRead}

	for name, ctx := range callers {
		for _, action := range actions {
			got, all, err := s.CallerProjects(ctx, action)
			require.NoError(t, err)
			var want []string
			for _, projectID := range []string{"p-hidden", "p-open", "p-other"} {
				if s.RequireProject(ctx, projectID, action) == nil {
					want = append(want, projectID)
				}
			}
			if all {
				assert.Len(t, want, 3, "%s %q", name, action)
				continue
			}
			assert.ElementsMatch(t, want, got, "%s %q", name, action)
		}
	}
}

type failingMembershipScopes struct{ fakeScopes }

func (failingMembershipScopes) WorkspaceIDsForUser(context.Context, string) ([]string, error) {
	return nil, apperrs.ErrConflict
}

func TestProjectsAnywhere_FailsWhenTheMembershipsCannotBeListed(t *testing.T) {
	s, _ := restrictedFixture(t)
	s.SetScopes(failingMembershipScopes{})

	_, _, err := s.ProjectsAnywhere(t.Context(), "team", permissions.TicketsRead)

	assert.ErrorIs(t, err, apperrs.ErrConflict)
}

type failingProjectScopes struct{ fakeScopes }

func (failingProjectScopes) ProjectIDs(context.Context, string) ([]string, error) {
	return nil, apperrs.ErrConflict
}

func TestProjectsWith_FailsWhenTheProjectsCannotBeListed(t *testing.T) {
	s, _ := restrictedFixture(t)
	s.SetScopes(failingProjectScopes{})

	_, err := s.ProjectsWith(t.Context(), "team", "ws", permissions.TicketsRead)

	assert.ErrorIs(t, err, apperrs.ErrConflict)
}
