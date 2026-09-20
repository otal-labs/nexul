package automations

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

func newTestService(repo Repo, perm PermissionGate) *Service {
	s := NewService(repo, perm)
	s.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestService_Create(t *testing.T) {
	t.Run("mints a custom automation with a token returned once", func(t *testing.T) {
		s := newTestService(newFakeRepo(), allowAll("owner"))
		a, token, err := s.Create(context.Background(), "owner", "My automation", []string{"tickets:write"})
		require.NoError(t, err)
		assert.Equal(t, "My automation", a.Name)
		assert.Equal(t, KindCustom, a.Kind)
		assert.False(t, a.Enabled)
		assert.True(t, strings.HasPrefix(token, tokenPrefix))
		assert.NotEmpty(t, a.TokenHash)
		assert.NotEmpty(t, a.TokenPrefix)
		assert.Nil(t, a.TokenRevokedAt)
	})
	t.Run("unknown scope is rejected when a resolver is wired", func(t *testing.T) {
		svc := NewService(newFakeRepo(), allowAll("creator-1"))
		svc.SetGateway(nil, func([]string) ([]string, error) { return nil, apperrs.ErrInvalid }, nil)
		_, _, err := svc.Create(context.Background(), "creator-1", "typo", []string{"ticket:write"})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("stored scopes are the resolver's effective set", func(t *testing.T) {
		svc := NewService(newFakeRepo(), allowAll("creator-1"))
		svc.SetGateway(nil, func([]string) ([]string, error) { return []string{"tickets:write", "tickets:read"}, nil }, nil)
		a, _, err := svc.Create(context.Background(), "creator-1", "expanded", []string{"tickets:write"})
		require.NoError(t, err)
		assert.Equal(t, []string{"tickets:write", "tickets:read"}, a.Scopes)
	})

	t.Run("no scopes is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), allowAll("owner"))
		_, _, err := s.Create(context.Background(), "owner", "x", nil)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty name is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), allowAll("owner"))
		_, _, err := s.Create(context.Background(), "owner", "  ", []string{"tickets:read"})
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("no actor is unauthorized", func(t *testing.T) {
		s := newTestService(newFakeRepo(), allowAll("owner"))
		_, _, err := s.Create(context.Background(), "", "x", []string{"tickets:read"})
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
	t.Run("actor without automations:write is forbidden", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakePerm(nil))
		_, _, err := s.Create(context.Background(), "alice", "x", []string{"tickets:read"})
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestService_ListAndGet(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	t.Run("list requires automations:read", func(t *testing.T) {
		_, err := s.List(context.Background(), "owner")
		require.NoError(t, err)
		_, err = s.List(context.Background(), "alice")
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
	t.Run("get returns the created automation", func(t *testing.T) {
		got, err := s.Get(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("an automation token reads without a creator holding automations:read", func(t *testing.T) {
		ctx := identity.WithActor(context.Background(), identity.Actor{Automation: &identity.AutomationRef{ID: created.ID}})
		got, err := s.Get(ctx, "", created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		_, err = s.SetEnabled(ctx, "", created.ID, true)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized, "only reads are identity; writes still need a permitted actor")
	})

	t.Run("get unknown id is not found", func(t *testing.T) {
		_, err := s.Get(context.Background(), "owner", "nope")
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestService_SetEnabled(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	require.False(t, created.Enabled)

	a, err := s.SetEnabled(context.Background(), "owner", created.ID, true)
	require.NoError(t, err)
	assert.True(t, a.Enabled)

	t.Run("no-op when already at the target state", func(t *testing.T) {
		a2, err := s.SetEnabled(context.Background(), "owner", created.ID, true)
		require.NoError(t, err)
		assert.Equal(t, a.UpdatedAt, a2.UpdatedAt)
	})
	t.Run("requires automations:write", func(t *testing.T) {
		_, err := s.SetEnabled(context.Background(), "alice", created.ID, false)
		assert.True(t, errors.Is(err, apperrs.ErrForbidden))
	})
}

func TestService_UpdateConfigValues(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	t.Run("replaces config values", func(t *testing.T) {
		a, err := s.UpdateConfigValues(context.Background(), "owner", created.ID, json.RawMessage(`{"status":"done"}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"status":"done"}`, string(a.ConfigValues))
	})
	t.Run("invalid JSON is rejected", func(t *testing.T) {
		_, err := s.UpdateConfigValues(context.Background(), "owner", created.ID, json.RawMessage(`{not json`))
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestService_Delete(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	require.NoError(t, s.Delete(context.Background(), "owner", created.ID))
	_, err = s.Get(context.Background(), "owner", created.ID)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}

func TestService_TokenLifecycle(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, firstToken, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	t.Run("mint rotates the token", func(t *testing.T) {
		a, secondToken, err := s.MintToken(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		assert.NotEqual(t, firstToken, secondToken)
		assert.Nil(t, a.TokenRevokedAt)
	})
	t.Run("revoke marks it revoked immediately", func(t *testing.T) {
		a, err := s.RevokeToken(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		require.NotNil(t, a.TokenRevokedAt)
	})
	t.Run("revoking an already-revoked token is a no-op", func(t *testing.T) {
		before, err := s.Get(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		after, err := s.RevokeToken(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		assert.Equal(t, before.TokenRevokedAt, after.TokenRevokedAt)
	})
	t.Run("minting again clears the revoked flag", func(t *testing.T) {
		a, _, err := s.MintToken(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		assert.Nil(t, a.TokenRevokedAt)
	})
}

func TestService_SyncFromCode(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "placeholder", []string{"tickets:read"})
	require.NoError(t, err)

	a, err := s.SyncFromCode(context.Background(), created.ID, "Ticket finished", "Moves finished tickets to done", []string{"ticket.finished"}, json.RawMessage(`{"status":{"type":"string"}}`))
	require.NoError(t, err)
	assert.Equal(t, "Ticket finished", a.Name)
	assert.Equal(t, "Moves finished tickets to done", a.Description)
	assert.Equal(t, []string{"ticket.finished"}, a.Subscriptions)
	assert.JSONEq(t, `{"status":{"type":"string"}}`, string(a.ConfigSchema))

	t.Run("empty schema keeps the existing one", func(t *testing.T) {
		a2, err := s.SyncFromCode(context.Background(), created.ID, "Ticket finished", "desc", []string{"ticket.finished"}, nil)
		require.NoError(t, err)
		assert.JSONEq(t, `{"status":{"type":"string"}}`, string(a2.ConfigSchema))
	})
	t.Run("blank name after sync is invalid", func(t *testing.T) {
		_, err := s.SyncFromCode(context.Background(), created.ID, "  ", "desc", nil, nil)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
}

func TestService_AuthenticateToken(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, token, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	t.Run("valid token resolves the automation", func(t *testing.T) {
		a, err := s.AuthenticateToken(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, created.ID, a.ID)
	})
	t.Run("empty token is unauthorized", func(t *testing.T) {
		_, err := s.AuthenticateToken(context.Background(), "")
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
	t.Run("unknown token is unauthorized", func(t *testing.T) {
		_, err := s.AuthenticateToken(context.Background(), "dat_bogus")
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
	t.Run("revoked token is unauthorized", func(t *testing.T) {
		_, err := s.RevokeToken(context.Background(), "owner", created.ID)
		require.NoError(t, err)
		_, err = s.AuthenticateToken(context.Background(), token)
		assert.True(t, errors.Is(err, apperrs.ErrUnauthorized))
	})
}

func TestService_RevokeToken_DisconnectsLiveConnection(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conns := &fakeConnRegistry{}
	s.SetConnectionRegistry(conns)

	_, err = s.RevokeToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)

	ids, reasons := conns.calledWith()
	require.Len(t, ids, 1)
	assert.Equal(t, created.ID, ids[0])
	assert.Equal(t, "token revoked", reasons[0])
}

func TestService_MintToken_DisconnectsLiveConnection(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conns := &fakeConnRegistry{}
	s.SetConnectionRegistry(conns)

	_, _, err = s.MintToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)

	ids, reasons := conns.calledWith()
	require.Len(t, ids, 1)
	assert.Equal(t, created.ID, ids[0])
	assert.Equal(t, "token rotated", reasons[0])
}

func TestService_RevokeToken_AlreadyRevoked_DoesNotDisconnectAgain(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conns := &fakeConnRegistry{}
	s.SetConnectionRegistry(conns)

	_, err = s.RevokeToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)
	_, err = s.RevokeToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)

	ids, _ := conns.calledWith()
	assert.Len(t, ids, 1)
}

func TestService_Delete_DisconnectsLiveConnection(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conns := &fakeConnRegistry{}
	s.SetConnectionRegistry(conns)

	require.NoError(t, s.Delete(context.Background(), "owner", created.ID))

	ids, reasons := conns.calledWith()
	require.Len(t, ids, 1)
	assert.Equal(t, created.ID, ids[0])
	assert.Equal(t, "automation deleted", reasons[0])
}

func TestService_RevokeToken_NoConnectionRegistry_DoesNotPanic(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	_, err = s.RevokeToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)
}

func TestService_SyncFromCode_UnchangedAnnounceDoesNotTouchUpdatedAt(t *testing.T) {
	s := newTestService(newFakeRepo(), allowAll("owner"))
	created, _, err := s.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	schema := json.RawMessage(`{"type":"object"}`)

	first := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return first }
	a, err := s.SyncFromCode(context.Background(), created.ID, "Synced", "desc", []string{"ticket.created"}, schema)
	require.NoError(t, err)
	require.Equal(t, first, a.UpdatedAt)

	s.now = func() time.Time { return first.Add(time.Hour) }
	again, err := s.SyncFromCode(context.Background(), created.ID, " Synced ", "desc", []string{"ticket.created"}, nil)
	require.NoError(t, err)
	assert.Equal(t, first, again.UpdatedAt, "a reconnect announcing the same code is not a change")

	changed, err := s.SyncFromCode(context.Background(), created.ID, "Synced", "new desc", []string{"ticket.created"}, nil)
	require.NoError(t, err)
	assert.Equal(t, first.Add(time.Hour), changed.UpdatedAt)
}
