package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/push"
)

func newSessionRecord(id, userID, hash string, at time.Time) *auth.Session {
	return &auth.Session{
		ID: id, UserID: userID, TokenHash: hash, Client: auth.ClientBrowser, Platform: "Linux", Label: "Chrome", IP: "10.0.0.1",
		CreatedAt: at, LastActiveAt: at, ExpiresAt: at.Add(30 * 24 * time.Hour),
	}
}

func TestSessionsRepo_Lifecycle(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(ctx, newTestUser("u2", "43", "other"))
	require.NoError(t, err)
	at := time.Unix(1_700_000_000, 0).UTC()

	_, err = s.Sessions.GetSessionByHash(ctx, "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	created := eventbus.OutboxEvent{ID: "evt-created", Topic: auth.TopicSessionCreated, Payload: auth.SessionChangedEvent{SessionID: "s1"}}
	require.NoError(t, s.Sessions.CreateSession(ctx, newSessionRecord("s1", "u1", "h1", at), created))
	require.NoError(t, s.Sessions.CreateSession(ctx, newSessionRecord("s2", "u1", "h2", at.Add(time.Minute))))
	require.NoError(t, s.Sessions.CreateSession(ctx, newSessionRecord("s3", "u2", "h3", at)))
	assert.Equal(t, 1, outboxCount(t, s, "evt-created"))
	err = s.Sessions.CreateSession(ctx, newSessionRecord("s4", "u1", "h1", at))
	require.ErrorIs(t, err, apperrs.ErrConflict, "a token hash is unique")

	got, err := s.Sessions.GetSessionByHash(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
	assert.Equal(t, auth.ClientBrowser, got.Client)
	assert.Equal(t, "Chrome", got.Label)
	assert.Equal(t, at, got.CreatedAt)

	list, err := s.Sessions.ListSessionsByUser(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "s2", list[0].ID, "most recently active first")

	require.NoError(t, s.Sessions.TouchSession(ctx, "s1", at.Add(2*time.Hour), "10.0.0.2", at.Add(48*time.Hour)))
	got, err = s.Sessions.GetSessionByHash(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, at.Add(2*time.Hour), got.LastActiveAt)
	assert.Equal(t, "10.0.0.2", got.IP)
	assert.Equal(t, at.Add(48*time.Hour), got.ExpiresAt)

	require.ErrorIs(t, s.Sessions.DeleteSession(ctx, "s1", "u2"), apperrs.ErrNotFound, "another user's session never deletes")
	revoked := eventbus.OutboxEvent{ID: "evt-revoked", Topic: auth.TopicSessionRevoked, Payload: auth.SessionChangedEvent{SessionID: "s1"}}
	require.NoError(t, s.Sessions.DeleteSession(ctx, "s1", "u1", revoked))
	assert.Equal(t, 1, outboxCount(t, s, "evt-revoked"))
	_, err = s.Sessions.GetSessionByHash(ctx, "h1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorIs(t, s.Sessions.DeleteSession(ctx, "s1", "u1"), apperrs.ErrNotFound, "a repeat is not found")
}

func TestSessionsRepo_DeleteOthersAndExpired(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(ctx, newTestUser("u2", "43", "other"))
	require.NoError(t, err)
	at := time.Unix(1_700_000_000, 0).UTC()
	for _, id := range []string{"s1", "s2", "s3"} {
		require.NoError(t, s.Sessions.CreateSession(ctx, newSessionRecord(id, "u1", "h-"+id, at)))
	}
	require.NoError(t, s.Sessions.CreateSession(ctx, newSessionRecord("o1", "u2", "h-o1", at)))

	evt := eventbus.OutboxEvent{ID: "evt-others", Topic: auth.TopicSessionRevoked, Payload: auth.SessionChangedEvent{SessionID: "s2"}}
	require.NoError(t, s.Sessions.DeleteOtherSessions(ctx, "u1", "s1", evt))
	assert.Equal(t, 1, outboxCount(t, s, "evt-others"))
	list, err := s.Sessions.ListSessionsByUser(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "s1", list[0].ID)
	others, err := s.Sessions.ListSessionsByUser(ctx, "u2")
	require.NoError(t, err)
	assert.Len(t, others, 1, "another user's sessions are untouched")

	expired := newSessionRecord("s9", "u1", "h-s9", at)
	expired.ExpiresAt = at.Add(time.Hour)
	require.NoError(t, s.Sessions.CreateSession(ctx, expired))
	require.NoError(t, s.Sessions.DeleteExpiredSessions(ctx, "u1", at.Add(time.Hour)))
	list, err = s.Sessions.ListSessionsByUser(ctx, "u1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "s1", list[0].ID, "only the row past its expiry is swept")
}

func TestSessionsRepo_PushTokens(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u1", "42", "onik97"))
	require.NoError(t, err)
	_, _, err = s.Users.UpsertUser(ctx, newTestUser("u2", "43", "other"))
	require.NoError(t, err)
	at := time.Now().UTC()
	phone := newSessionRecord("p1", "u1", "h-p1", at)
	phone.Client = auth.ClientPhone
	require.NoError(t, s.Sessions.CreateSession(ctx, phone))
	require.NoError(t, s.Sessions.CreateSession(ctx, newSessionRecord("b1", "u1", "h-b1", at)))
	expired := newSessionRecord("p2", "u1", "h-p2", at.Add(-48*time.Hour))
	expired.Client = auth.ClientPhone
	expired.ExpiresAt = at.Add(-time.Hour)
	require.NoError(t, s.Sessions.CreateSession(ctx, expired))
	other := newSessionRecord("p3", "u2", "h-p3", at)
	other.Client = auth.ClientPhone
	require.NoError(t, s.Sessions.CreateSession(ctx, other))

	require.ErrorIs(t, s.Sessions.SetSessionPushToken(ctx, "p1", "u2", "tok"), apperrs.ErrNotFound, "another user's session never takes a token")
	require.NoError(t, s.Sessions.SetSessionPushToken(ctx, "p1", "u1", "tok-phone"))
	require.NoError(t, s.Sessions.SetSessionPushToken(ctx, "b1", "u1", "tok-browser"))
	require.NoError(t, s.Sessions.SetSessionPushToken(ctx, "p2", "u1", "tok-expired"))
	require.NoError(t, s.Sessions.SetSessionPushToken(ctx, "p3", "u2", "tok-other"))

	targets, err := s.Sessions.ListPushTargets(ctx, []string{"u1"})
	require.NoError(t, err)
	assert.Equal(t, []push.Target{{SessionID: "p1", UserID: "u1", Token: "tok-phone"}}, targets, "browsers and expired phones never receive a push")

	none, err := s.Sessions.ListPushTargets(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, none)

	require.NoError(t, s.Sessions.ClearPushToken(ctx, "p1", "u1"))
	require.NoError(t, s.Sessions.ClearPushToken(ctx, "missing", "u1"), "a row already signed out is not an error")
	targets, err = s.Sessions.ListPushTargets(ctx, []string{"u1", "u2"})
	require.NoError(t, err)
	assert.Equal(t, []push.Target{{SessionID: "p3", UserID: "u2", Token: "tok-other"}}, targets)

	require.NoError(t, s.Sessions.DeleteSession(ctx, "p3", "u2"))
	targets, err = s.Sessions.ListPushTargets(ctx, []string{"u1", "u2"})
	require.NoError(t, err)
	assert.Empty(t, targets, "signing out removes the row and its token with it")
}
