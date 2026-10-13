package storage

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// newSharingStore is sam's laptop with a rule naming bob, both in the default workspace, and lena in none of sam's.
func newSharingStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	db := newTestDB(t)
	s := New(db, testEncKey)
	ctx := context.Background()
	for _, u := range []string{"sam", "bob", "lena"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser(u, u, u))
		require.NoError(t, err)
	}
	_, err := db.Exec(`INSERT INTO workspaces (id, name, slug, created_at, updated_at) VALUES ('ws-other', 'Other', 'other', 1, 1);
INSERT INTO workspace_members (user_id, workspace_id, created_at) VALUES ('sam', 'workspace-default', 1), ('bob', 'workspace-default', 1), ('lena', 'ws-other', 1);`)
	require.NoError(t, err)
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("laptop", "sam", "laptop")))
	require.NoError(t, s.Access.Set(ctx, "computer", "laptop", "bob", permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands), nil))
	return s, db
}

func TestAccessRepo_ComputerAccess_ReadsTheOwnerAndOnlyALiveRule(t *testing.T) {
	t.Parallel()
	s, db := newSharingStore(t)
	ctx := context.Background()

	owner, allow, err := s.Access.ComputerAccess(ctx, "laptop", "sam")
	require.NoError(t, err)
	assert.Equal(t, "sam", owner)
	assert.Empty(t, allow)
	owner, allow, err = s.Access.ComputerAccess(ctx, "laptop", "bob")
	require.NoError(t, err)
	assert.Equal(t, "sam", owner)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands), allow)
	_, _, err = s.Access.ComputerAccess(ctx, "laptop", "lena")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "no rule names her")
	_, _, err = s.Access.ComputerAccess(ctx, "gone", "sam")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	_, err = db.Exec(`UPDATE users SET account_status = 'disabled' WHERE id = 'sam'`)
	require.NoError(t, err)
	_, _, err = s.Access.ComputerAccess(ctx, "laptop", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a disabled owner shares nothing")
	_, _, err = s.Access.ComputerAccess(ctx, "laptop", "sam")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a disabled account opens nothing, its own computer included")

	_, err = db.Exec(`UPDATE users SET account_status = 'active' WHERE id = 'sam'; DELETE FROM workspace_members WHERE user_id = 'bob'`)
	require.NoError(t, err)
	_, _, err = s.Access.ComputerAccess(ctx, "laptop", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "no shared workspace")

	shares, err := s.Access.SharesWithOwner(ctx, "sam", "bob")
	require.NoError(t, err)
	assert.False(t, shares)
	shares, err = s.Access.SharesWithOwner(ctx, "sam", "sam")
	require.NoError(t, err)
	assert.True(t, shares)
}

func TestAccessRepo_EndComputerRules_DeletesOnlyTheEndedWithAnEventEach(t *testing.T) {
	t.Parallel()
	s, db := newSharingStore(t)
	ctx := context.Background()
	require.NoError(t, s.Access.Set(ctx, "computer", "laptop", "lena", permissions.SetOf(permissions.ComputersSee), nil))

	var ended []string
	event := func(computerID, ownerID, userID string) eventbus.OutboxEvent {
		ended = append(ended, computerID+"/"+ownerID+"/"+userID)
		return eventbus.OutboxEvent{ID: "evt-" + userID, Topic: "computer.grant_changed", Payload: map[string]string{"user_id": userID}}
	}
	require.NoError(t, s.Access.EndComputerRules(ctx, event))
	assert.Equal(t, []string{"laptop/sam/lena"}, ended)
	_, err := s.Access.Get(ctx, "computer", "laptop", "lena")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.Access.Get(ctx, "computer", "laptop", "bob")
	require.NoError(t, err, "bob's rule is live")
	var rows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE id = 'evt-lena'`).Scan(&rows))
	assert.Equal(t, 1, rows)

	ended = nil
	require.NoError(t, s.Access.EndComputerRules(ctx, event))
	assert.Empty(t, ended, "safe to run again")
}

func TestPairingRepo_ListComputersFor_OwnThenShared(t *testing.T) {
	t.Parallel()
	s, db := newSharingStore(t)
	ctx := context.Background()
	require.NoError(t, s.Pairing.SaveComputer(ctx, newTestComputer("desk", "bob", "desk")))

	got, err := s.Pairing.ListComputersFor(ctx, "bob")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "desk", got[0].ID)
	assert.Nil(t, got[0].Shared)
	assert.Equal(t, "laptop", got[1].ID)
	require.NotNil(t, got[1].Shared)
	assert.Equal(t, "sam", got[1].Shared.OwnerID)
	assert.Equal(t, "Name sam", got[1].Shared.OwnerName)
	assert.Equal(t, permissions.SetOf(permissions.ComputersSee, permissions.ComputersCommands), got[1].Shared.Permissions)

	got, err = s.Pairing.ListComputersFor(ctx, "sam")
	require.NoError(t, err)
	require.Len(t, got, 1, "the owner's own, once, whoever it is shared with")

	require.NoError(t, s.Access.Set(ctx, "computer", "laptop", "bob", permissions.SetOf(permissions.ComputersCommands), nil))
	got, err = s.Pairing.ListComputersFor(ctx, "bob")
	require.NoError(t, err)
	assert.Len(t, got, 1, "a rule without See lists nothing")

	require.NoError(t, s.Access.Set(ctx, "computer", "laptop", "bob", permissions.SetOf(permissions.ComputersSee), nil))
	_, err = db.Exec(`DELETE FROM pairing_computers WHERE id = 'laptop'`)
	require.NoError(t, err)
	_, err = s.Access.Get(ctx, "computer", "laptop", "bob")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a computer's rules go with it")
}
