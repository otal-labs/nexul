package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func seedTestAutomation(t *testing.T, s *Store, id string) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, s.Automations.Create(context.Background(), &automations.Automation{
		ID: id, Name: "test", Kind: automations.KindCustom, Scopes: []string{"x"},
		TokenHash: "h", CreatedAt: now, UpdatedAt: now,
	}))
}

func TestAutomationVersionsRepo_InsertActive(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")

	v := &automations.Version{ID: "v1", AutomationID: "a1", Code: "code v1", PusherID: "seed", Status: automations.VersionActive, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.InsertActive(context.Background(), v))
	assert.Equal(t, 1, v.Sequence, "the first version for an automation gets sequence 1")

	got, err := s.AutomationVersions.Active(context.Background(), "a1")
	require.NoError(t, err)
	assert.Equal(t, "code v1", got.Code)
	assert.Equal(t, automations.VersionActive, got.Status)
}

func TestAutomationVersionsRepo_ReplacePending(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")

	v1 := &automations.Version{ID: "v1", AutomationID: "a1", Code: "v1", PusherID: "owner", Status: automations.VersionPending, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.ReplacePending(context.Background(), v1))
	assert.Equal(t, 1, v1.Sequence)

	v2 := &automations.Version{ID: "v2", AutomationID: "a1", Code: "v2", PusherID: "owner", Status: automations.VersionPending, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.ReplacePending(context.Background(), v2))
	assert.Equal(t, 2, v2.Sequence, "sequence still advances even though the prior pending row is discarded")

	_, err := s.AutomationVersions.Get(context.Background(), "a1", v1.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "the replaced pending version is gone, not just superseded")

	pending, err := s.AutomationVersions.Pending(context.Background(), "a1")
	require.NoError(t, err)
	assert.Equal(t, v2.ID, pending.ID)

	list, err := s.AutomationVersions.ListByAutomation(context.Background(), "a1")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestAutomationVersionsRepo_Activate(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")

	v1 := &automations.Version{ID: "v1", AutomationID: "a1", Code: "v1", PusherID: "owner", Status: automations.VersionPending, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.ReplacePending(context.Background(), v1))

	activated, err := s.AutomationVersions.Activate(context.Background(), "a1", v1.ID)
	require.NoError(t, err)
	assert.Equal(t, automations.VersionActive, activated.Status)

	v2 := &automations.Version{ID: "v2", AutomationID: "a1", Code: "v2", PusherID: "owner", Status: automations.VersionPending, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.ReplacePending(context.Background(), v2))
	activated2, err := s.AutomationVersions.Activate(context.Background(), "a1", v2.ID)
	require.NoError(t, err)
	assert.Equal(t, automations.VersionActive, activated2.Status)

	old, err := s.AutomationVersions.Get(context.Background(), "a1", v1.ID)
	require.NoError(t, err)
	assert.Equal(t, automations.VersionInactive, old.Status, "repointing active moves the prior active version to inactive, not deleting it")

	// Rollback is the same operation, just aimed at an older (inactive) version.
	rolledBack, err := s.AutomationVersions.Activate(context.Background(), "a1", v1.ID)
	require.NoError(t, err)
	assert.Equal(t, automations.VersionActive, rolledBack.Status)

	t.Run("activating a version that doesn't belong to the automation 404s", func(t *testing.T) {
		seedTestAutomation(t, s, "a2")
		_, err := s.AutomationVersions.Activate(context.Background(), "a2", v1.ID)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("activating an unknown version id 404s", func(t *testing.T) {
		_, err := s.AutomationVersions.Activate(context.Background(), "a1", "missing")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestAutomationVersionsRepo_PendingAndActive_NotFound(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")

	_, err := s.AutomationVersions.Pending(context.Background(), "a1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = s.AutomationVersions.Active(context.Background(), "a1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestAutomationVersionsRepo_OnePendingOnePerAutomation_Enforced(t *testing.T) {
	// Guards the migration's partial unique indexes directly: even a
	// hand-rolled INSERT that bypasses ReplacePending's delete-then-insert
	// must fail, proving the DB — not just app logic — enforces the
	// invariant.
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")
	now := time.Now().Unix()
	_, err := s.db.Exec(`INSERT INTO automation_versions (id, automation_id, sequence, code, pusher_id, message, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"v1", "a1", 1, "code", "owner", "", "pending", now)
	require.NoError(t, err)

	_, err = s.db.Exec(`INSERT INTO automation_versions (id, automation_id, sequence, code, pusher_id, message, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"v2", "a1", 2, "code2", "owner", "", "pending", now)
	require.Error(t, err, "a second pending row for the same automation must violate the unique index")
}

func TestAutomationVersionsRepo_ListByAutomation_NewestFirst(t *testing.T) {
	s := newTestStore(t)
	seedTestAutomation(t, s, "a1")

	v1 := &automations.Version{ID: "v1", AutomationID: "a1", Code: "v1", PusherID: "owner", Status: automations.VersionActive, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.InsertActive(context.Background(), v1))
	_, err := s.AutomationVersions.Activate(context.Background(), "a1", v1.ID)
	require.NoError(t, err)
	v2 := &automations.Version{ID: "v2", AutomationID: "a1", Code: "v2", PusherID: "owner", Status: automations.VersionPending, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.AutomationVersions.ReplacePending(context.Background(), v2))

	list, err := s.AutomationVersions.ListByAutomation(context.Background(), "a1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "v2", list[0].ID, "newest sequence first")
	assert.Equal(t, "v1", list[1].ID)
}
