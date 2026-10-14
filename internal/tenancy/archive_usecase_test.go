package tenancy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newArchivingService(repo *fakeRepo) *Service {
	wsPerms := newFakeWorkspacePermissionGate()
	wsPerms.perms["u-1"] = []string{"workspaces:delete"}
	s := NewService(repo, repo, newFakeInviteRepo(), &fakeRoleGate{}, newFakePermissionGate(), newFakeRoleNameGate(), wsPerms, newFakeAllowlistGate(), newFakeUserLookupGate(), &fakeChannelGate{}, &fakeDefaultsGate{}, newFakeAccountGate())
	s.now = func() time.Time { return fixedNow }
	return s
}

func TestArchiveAndRestore_RepeatingTheStateChangesAndPublishesNothing(t *testing.T) {
	repo := newFakeRepo()
	repo.workspaces["ws-acme"] = &Workspace{ID: "ws-acme", Name: "Acme", Slug: "acme"}
	repo.members["ws-acme"] = []string{"u-1", "u-2"}
	s := newArchivingService(repo)
	ctx := context.Background()

	_, err := s.Restore(ctx, "u-1", "ws-acme")
	require.NoError(t, err)
	assert.Empty(t, repo.events, "a live workspace is already restored")

	_, err = s.Archive(ctx, "u-1", "ws-acme")
	require.NoError(t, err)
	_, err = s.Archive(ctx, "u-1", "ws-acme")
	require.NoError(t, err)
	require.Len(t, repo.events, 1, "the second archive finds it archived")
	assert.Equal(t, TopicWorkspaceArchived, repo.events[0].Topic)
	assert.ElementsMatch(t, []string{"u-1", "u-2"}, repo.events[0].Payload.(WorkspaceArchiveEvent).UserIDs)
}
