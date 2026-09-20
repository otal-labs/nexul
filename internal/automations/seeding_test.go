package automations

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSeeder(automationsRepo Repo, versionsRepo VersionsRepo, defs []DefaultDefinition) *Seeder {
	s := NewSeeder(automationsRepo, newTestVersionsService(automationsRepo, versionsRepo, allowAll("system")), defs, slog.Default())
	s.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestSeeder_Seed(t *testing.T) {
	def := DefaultDefinition{
		ID:          "default-board-pair-1",
		Name:        "Ticket finished",
		Description: "Moves a ticket to done when every linked PR merges.",
		Code:        "export default { on: () => {} }",
		Scopes:      []string{"tickets:write"},
		Enabled:     true,
	}

	t.Run("creates the automation record on a fresh instance", func(t *testing.T) {
		autoRepo := newFakeRepo()
		s := newTestSeeder(autoRepo, newFakeVersionsRepo(), []DefaultDefinition{def})
		require.NoError(t, s.Seed(context.Background()))

		a, err := autoRepo.Get(context.Background(), def.ID)
		require.NoError(t, err)
		assert.Equal(t, KindDefault, a.Kind)
		assert.True(t, a.Enabled)
		assert.Equal(t, def.Name, a.Name)
	})

	t.Run("the first seed activates the code directly, no manual merge needed", func(t *testing.T) {
		autoRepo := newFakeRepo()
		vRepo := newFakeVersionsRepo()
		s := newTestSeeder(autoRepo, vRepo, []DefaultDefinition{def})
		require.NoError(t, s.Seed(context.Background()))

		active, err := vRepo.Active(context.Background(), def.ID)
		require.NoError(t, err)
		assert.Equal(t, def.Code, active.Code)
		assert.Equal(t, seedPusherID, active.PusherID)

		_, err = vRepo.Pending(context.Background(), def.ID)
		require.Error(t, err, "no pending version — the first seed skips the review step entirely")
	})

	t.Run("re-seeding unchanged code is idempotent: no new version, no automation churn", func(t *testing.T) {
		autoRepo := newFakeRepo()
		vRepo := newFakeVersionsRepo()
		s := newTestSeeder(autoRepo, vRepo, []DefaultDefinition{def})
		require.NoError(t, s.Seed(context.Background()))
		require.NoError(t, s.Seed(context.Background()))

		list, err := vRepo.ListByAutomation(context.Background(), def.ID)
		require.NoError(t, err)
		assert.Len(t, list, 1, "seeding the same code twice must not create a second version")
	})

	t.Run("an existing default with stale scopes is brought up to the definition", func(t *testing.T) {
		autoRepo := newFakeRepo()
		vRepo := newFakeVersionsRepo()
		s := newTestSeeder(autoRepo, vRepo, []DefaultDefinition{def})
		require.NoError(t, s.Seed(context.Background()))
		stale, err := autoRepo.Get(context.Background(), def.ID)
		require.NoError(t, err)
		stale.Scopes = []string{"ticket.update"}
		require.NoError(t, autoRepo.Update(context.Background(), stale))

		require.NoError(t, s.Seed(context.Background()))
		a, err := autoRepo.Get(context.Background(), def.ID)
		require.NoError(t, err)
		assert.Equal(t, def.Scopes, a.Scopes)
	})

	t.Run("an upgrade with different code lands pending, and never touches enabled/config", func(t *testing.T) {
		autoRepo := newFakeRepo()
		vRepo := newFakeVersionsRepo()
		s := newTestSeeder(autoRepo, vRepo, []DefaultDefinition{def})
		require.NoError(t, s.Seed(context.Background()))

		// Simulate the owner's own edits surviving an upgrade (enable
		// state and config values are never touched by seeding).
		a, err := autoRepo.Get(context.Background(), def.ID)
		require.NoError(t, err)
		a.Enabled = false
		a.ConfigValues = []byte(`{"channel":"#deploys"}`)
		require.NoError(t, autoRepo.Update(context.Background(), a))

		upgraded := def
		upgraded.Code = "export default { on: () => { /* v2 */ } }"
		s2 := newTestSeeder(autoRepo, vRepo, []DefaultDefinition{upgraded})
		require.NoError(t, s2.Seed(context.Background()))

		pending, err := vRepo.Pending(context.Background(), def.ID)
		require.NoError(t, err)
		assert.Equal(t, upgraded.Code, pending.Code)

		active, err := vRepo.Active(context.Background(), def.ID)
		require.NoError(t, err)
		assert.Equal(t, def.Code, active.Code, "the upgrade never silently replaces the running code")

		reloaded, err := autoRepo.Get(context.Background(), def.ID)
		require.NoError(t, err)
		assert.False(t, reloaded.Enabled, "seeding never touches enable state")
		assert.JSONEq(t, `{"channel":"#deploys"}`, string(reloaded.ConfigValues), "seeding never touches config values")
	})

	t.Run("an empty definition set seeds nothing (ticket 12 wires the real bundles)", func(t *testing.T) {
		autoRepo := newFakeRepo()
		s := newTestSeeder(autoRepo, newFakeVersionsRepo(), nil)
		require.NoError(t, s.Seed(context.Background()))
		list, err := autoRepo.List(context.Background())
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("a real Get failure (not just 'missing') aborts the seed", func(t *testing.T) {
		repo := newErroringRepo()
		repo.getErr = errBoom
		s := newTestSeeder(repo, newFakeVersionsRepo(), []DefaultDefinition{def})
		require.ErrorIs(t, s.Seed(context.Background()), errBoom)
	})

	t.Run("a version-seeding failure aborts the seed", func(t *testing.T) {
		autoRepo := newFakeRepo()
		vRepo := newErroringVersionsRepo()
		vRepo.insertActiveErr = errBoom
		s := newTestSeeder(autoRepo, vRepo, []DefaultDefinition{def})
		require.ErrorIs(t, s.Seed(context.Background()), errBoom)
	})

	t.Run("a definition with no scopes fails loudly rather than minting an unusable token", func(t *testing.T) {
		autoRepo := newFakeRepo()
		bad := def
		bad.Scopes = nil
		s := newTestSeeder(autoRepo, newFakeVersionsRepo(), []DefaultDefinition{bad})
		require.Error(t, s.Seed(context.Background()))
	})

	t.Run("a definition with no name fails Automation.Validate", func(t *testing.T) {
		autoRepo := newFakeRepo()
		bad := def
		bad.Name = "   "
		s := newTestSeeder(autoRepo, newFakeVersionsRepo(), []DefaultDefinition{bad})
		require.Error(t, s.Seed(context.Background()))
	})

	t.Run("an automations.Create failure aborts the seed", func(t *testing.T) {
		repo := newErroringRepo()
		repo.createErr = errBoom
		s := newTestSeeder(repo, newFakeVersionsRepo(), []DefaultDefinition{def})
		require.ErrorIs(t, s.Seed(context.Background()), errBoom)
	})
}

func TestNewSeeder_NilLoggerDefaults(t *testing.T) {
	s := NewSeeder(newFakeRepo(), newTestVersionsService(newFakeRepo(), newFakeVersionsRepo(), allowAll("system")), nil, nil)
	assert.NotNil(t, s.log)
}
