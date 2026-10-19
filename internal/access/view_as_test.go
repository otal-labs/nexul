package access

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// viewing is ctx acting as userID for bob, who views as them in ws-a alone.
func viewing(ctx context.Context, userID string) context.Context {
	return identity.WithActor(ctx, identity.Actor{ID: userID, Viewer: &identity.ViewerRef{ID: "bob", WorkspaceID: "ws-a"}})
}

func TestViewAs_HoldsMembershipToThePinnedWorkspaceWithoutPoisoningTheMemo(t *testing.T) {
	roles := newFakeRoles()
	roles.set("ws-a", "alice", RoleInfo{})
	roles.set("ws-b", "alice", RoleInfo{})
	s := newService(newFakeRepo(), newFakeUsers())
	s.SetRoles(roles)
	s.SetScopes(fakeScopes{workspaces: map[string][]string{"alice": {"ws-a", "ws-b"}}})
	plain := WithMemo(context.Background(), s.NewMemo())
	view := viewing(plain, "alice")

	viewed, err := s.WorkspacesOf(view, "alice")
	require.NoError(t, err)
	assert.Equal(t, []string{"ws-a"}, viewed, "the list holds only the pinned workspace")
	require.ErrorIs(t, s.Require(view, "ws-b", permissions.Member), apperrs.ErrNotFound)
	require.NoError(t, s.Require(view, "ws-a", permissions.Member))

	own, err := s.WorkspacesOf(plain, "alice")
	require.NoError(t, err)
	assert.Equal(t, []string{"ws-a", "ws-b"}, own, "the same request's memo still holds the person's own list")
	require.NoError(t, s.Require(identity.WithActor(plain, identity.Actor{ID: "alice"}), "ws-b", permissions.Member))
}

func TestViewAs_AnswersNoComputerPermissions(t *testing.T) {
	s, _, _ := newComputerService(t)

	owner, _, err := s.ComputerPermissions(context.Background(), "sam", "laptop")
	require.NoError(t, err)
	assert.Equal(t, "sam", owner)

	_, _, err = s.ComputerPermissions(viewing(context.Background(), "sam"), "sam", "laptop")
	assert.ErrorIs(t, err, apperrs.ErrNotFound, "a computer is private to its owner even from someone viewing as them")
}
