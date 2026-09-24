package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newComputerPAT(id, hash, computerID string) *auth.PersonalAccessToken {
	p := newPATRecord("u1", "Nexul MCP on Laptop", hash, id, time.Unix(100, 0).UTC())
	p.ComputerID = computerID
	return p
}

func outboxCount(t *testing.T, s *Store, id string) int {
	t.Helper()
	var n int
	require.NoError(t, s.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM outbox WHERE id = ?`, id).Scan(&n))
	return n
}

func TestPATsRepo_GetActiveForComputer(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)

	_, err = s.PATs.GetActiveForComputer(ctx, "u1", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	minted := eventbus.OutboxEvent{ID: "evt-minted", Topic: auth.TopicTokenMinted, Payload: auth.TokenChangedEvent{TokenID: "p1"}}
	require.NoError(t, s.PATs.Create(ctx, newComputerPAT("p1", "hash-1", "c1"), minted))
	require.NoError(t, s.PATs.Create(ctx, newPATRecord("u1", "ci", "hash-2", "p2", time.Unix(100, 0).UTC())))
	assert.Equal(t, 1, outboxCount(t, s, "evt-minted"))

	got, err := s.PATs.GetActiveForComputer(ctx, "u1", "c1")
	require.NoError(t, err)
	assert.Equal(t, "p1", got.ID)
	assert.Equal(t, "c1", got.ComputerID)
	_, err = s.PATs.GetActiveForComputer(ctx, "u2", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "another user's computer token never resolves")
	_, err = s.PATs.GetActiveForComputer(ctx, "u1", "")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a token the user named is no computer's")

	revoked := eventbus.OutboxEvent{ID: "evt-revoked", Topic: auth.TopicTokenRevoked, Payload: auth.TokenChangedEvent{TokenID: "p1"}}
	require.NoError(t, s.PATs.Revoke(ctx, "p1", "u1", revoked))
	assert.Equal(t, 1, outboxCount(t, s, "evt-revoked"))
	_, err = s.PATs.GetActiveForComputer(ctx, "u1", "c1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPATsRepo_OneActiveTokenPerComputer(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.PATs.Create(ctx, newComputerPAT("p1", "hash-1", "c1")))

	evt := eventbus.OutboxEvent{ID: "evt-second", Topic: auth.TopicTokenMinted, Payload: auth.TokenChangedEvent{TokenID: "p2"}}
	require.ErrorIs(t, s.PATs.Create(ctx, newComputerPAT("p2", "hash-2", "c1"), evt), apperrs.ErrConflict)
	assert.Equal(t, 0, outboxCount(t, s, "evt-second"), "a rejected token writes no event")

	require.NoError(t, s.PATs.Revoke(ctx, "p1", "u1"))
	require.NoError(t, s.PATs.Create(ctx, newComputerPAT("p2", "hash-2", "c1")), "a revoked token frees the computer's slot")
}

func TestPATsRepo_Revoke_AlreadyRevokedWritesNoEvent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	require.NoError(t, s.PATs.Create(ctx, newComputerPAT("p1", "hash-1", "c1")))
	require.NoError(t, s.PATs.Revoke(ctx, "p1", "u1"))

	evt := eventbus.OutboxEvent{ID: "evt-again", Topic: auth.TopicTokenRevoked, Payload: auth.TokenChangedEvent{TokenID: "p1"}}
	require.ErrorIs(t, s.PATs.Revoke(ctx, "p1", "u1", evt), apperrs.ErrNotFound)
	assert.Equal(t, 0, outboxCount(t, s, "evt-again"))
}
