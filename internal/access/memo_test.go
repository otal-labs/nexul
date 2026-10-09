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
