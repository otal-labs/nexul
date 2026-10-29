package storage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

func newTestAutomation() *automations.Automation {
	return &automations.Automation{
		ID:            "auto-1",
		Name:          "Ticket finished",
		Description:   "Moves finished tickets to done",
		Kind:          automations.KindCustom,
		Enabled:       true,
		Subscriptions: []string{"ticket.finished"},
		ConfigSchema:  json.RawMessage(`{"status":{"type":"string"}}`),
		ConfigValues:  json.RawMessage(`{"status":"done"}`),
		Scopes:        []string{"tickets:write"},
		TokenHash:     "hash-1",
		TokenPrefix:   "aB3xY9",
		CreatedAt:     fixedNow,
		UpdatedAt:     fixedNow,
	}
}

func TestAutomationsRepo_CreateGet_RoundTrip(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	a := newTestAutomation()

	require.NoError(t, s.Automations.Create(ctx, a))
	got, err := s.Automations.Get(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.Name, got.Name)
	assert.Equal(t, a.Description, got.Description)
	assert.Equal(t, a.Kind, got.Kind)
	assert.True(t, got.Enabled)
	assert.Equal(t, a.Subscriptions, got.Subscriptions)
	assert.JSONEq(t, string(a.ConfigSchema), string(got.ConfigSchema))
	assert.JSONEq(t, string(a.ConfigValues), string(got.ConfigValues))
	assert.Equal(t, a.Scopes, got.Scopes)
	assert.Equal(t, a.TokenHash, got.TokenHash)
	assert.Equal(t, a.TokenPrefix, got.TokenPrefix)
	assert.Nil(t, got.TokenRevokedAt)
}

func TestAutomationsRepo_GetByTokenHash_Found(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	a := newTestAutomation()
	require.NoError(t, s.Automations.Create(ctx, a))

	got, err := s.Automations.GetByTokenHash(ctx, a.TokenHash)
	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)
}

func TestAutomationsRepo_List_OrderedByCreatedAt(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	first := newTestAutomation()
	first.ID = "auto-1"
	first.CreatedAt = fixedNow
	second := newTestAutomation()
	second.ID = "auto-2"
	second.CreatedAt = fixedNow.Add(time.Second)
	require.NoError(t, s.Automations.Create(ctx, second))
	require.NoError(t, s.Automations.Create(ctx, first))

	list, err := s.Automations.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "auto-1", list[0].ID)
	assert.Equal(t, "auto-2", list[1].ID)
}

func TestAutomationsRepo_Update(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	a := newTestAutomation()
	require.NoError(t, s.Automations.Create(ctx, a))

	t.Run("round trips a revoked token and disabled state", func(t *testing.T) {
		revokedAt := fixedNow.Add(time.Minute)
		a.Enabled = false
		a.TokenHash = "hash-2"
		a.TokenPrefix = "zZ9kQ1"
		a.TokenRevokedAt = &revokedAt
		a.UpdatedAt = revokedAt
		require.NoError(t, s.Automations.Update(ctx, a))

		got, err := s.Automations.Get(ctx, a.ID)
		require.NoError(t, err)
		assert.False(t, got.Enabled)
		assert.Equal(t, "hash-2", got.TokenHash)
		require.NotNil(t, got.TokenRevokedAt)
		assert.True(t, got.TokenRevokedAt.Equal(revokedAt))
	})
	t.Run("unknown id is not found", func(t *testing.T) {
		other := newTestAutomation()
		other.ID = "nope"
		assert.True(t, errors.Is(s.Automations.Update(ctx, other), apperrs.ErrNotFound))
	})
}

func TestAutomationsRepo_Delete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	a := newTestAutomation()
	require.NoError(t, s.Automations.Create(ctx, a))

	require.NoError(t, s.Automations.Delete(ctx, a.ID))
	_, err := s.Automations.Get(ctx, a.ID)
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))

	t.Run("deleting again is not found", func(t *testing.T) {
		assert.True(t, errors.Is(s.Automations.Delete(ctx, a.ID), apperrs.ErrNotFound))
	})
}

func TestToAutomation_MalformedListColumn_ReturnsError(t *testing.T) {
	_, err := toAutomation(sqlcgen.Automation{ID: "auto-bad", Subscriptions: []byte(`{`), Scopes: []byte(`[]`)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode subscriptions for automation auto-bad")
}

func TestToAutomation_NilListColumns_DecodeAsEmpty(t *testing.T) {
	got, err := toAutomation(sqlcgen.Automation{ID: "auto-nil", Subscriptions: nil, Scopes: nil})
	require.NoError(t, err)
	assert.Empty(t, got.Subscriptions)
	assert.Empty(t, got.Scopes)
}

func TestAutomationsRepo_List_EmptyListColumns_ListsWithEmptyLists(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	_, err := s.db.ExecContext(ctx, `INSERT INTO automations
		(id, name, kind, subscriptions, config_schema, config_values, scopes, token_hash, created_at, updated_at)
		VALUES ('auto-empty', 'Empty lists', 'custom', x'', '{}', '{}', x'', 'hash-empty', 0, 0)`)
	require.NoError(t, err)

	got, err := s.Automations.List(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "auto-empty", got[0].ID)
	assert.Empty(t, got[0].Subscriptions)
	assert.Empty(t, got[0].Scopes)
}

func TestAutomationsRepo_ListHealth_ReadsLastRunWeekCountsAndPendingVersion(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	seedTestAutomation(t, s, "busy")
	seedTestAutomation(t, s, "idle")
	now := time.Now().UTC().Truncate(time.Second)

	old := newTestRun("old", "busy", now.Add(-8*24*time.Hour))
	old.Outcome = automations.RunOutcomeFailure
	ok := newTestRun("ok", "busy", now.Add(-2*time.Hour))
	crash := newTestRun("crash", "busy", now.Add(-time.Hour))
	crash.Outcome = automations.RunOutcomeCrash
	crash.Error = "handler exceeded the 30 second limit"
	for _, r := range []*automations.Run{old, ok, crash} {
		require.NoError(t, s.AutomationRuns.Create(ctx, r))
	}
	pending := &automations.Version{ID: "p", AutomationID: "busy", Code: "c", PusherID: "owner", Status: automations.VersionPending, CreatedAt: now}
	require.NoError(t, s.AutomationVersions.ReplacePending(ctx, pending))

	health, err := s.Automations.ListHealth(ctx, []string{"busy", "idle"}, now.Add(-7*24*time.Hour))
	require.NoError(t, err)

	busy := health["busy"]
	require.NotNil(t, busy.LastRun)
	assert.Equal(t, automations.RunOutcomeCrash, busy.LastRun.Outcome)
	assert.Equal(t, now.Add(-time.Hour), busy.LastRun.StartedAt)
	assert.Equal(t, "handler exceeded the 30 second limit", busy.LastRun.Error)
	assert.Equal(t, automations.RunCounts{Total: 2, Failed: 1}, busy.Runs7d, "the 8-day-old failure is outside the window")
	require.NotNil(t, busy.PendingVersion)
	assert.Equal(t, pending.Sequence, *busy.PendingVersion)

	idle := health["idle"]
	assert.Nil(t, idle.LastRun)
	assert.Nil(t, idle.PendingVersion)
	assert.Equal(t, automations.RunCounts{}, idle.Runs7d)
}

func TestAutomationsRepo_ListHealth_NoIDs_ReturnsEmpty(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")

	health, err := s.Automations.ListHealth(context.Background(), nil, time.Now())
	require.NoError(t, err)
	assert.Empty(t, health)
}
