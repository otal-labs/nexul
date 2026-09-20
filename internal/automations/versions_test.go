package automations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestVersionsService(automationsRepo Repo, versionsRepo VersionsRepo, perm PermissionGate) *VersionsService {
	s := NewVersionsService(versionsRepo, automationsRepo, perm)
	s.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	return s
}

// seedAutomation creates a bare custom automation directly through the repo,
// bypassing Service.Create's permission gate — the version tests only care
// that an automation with this id exists.
func seedAutomation(t *testing.T, repo Repo, id string) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), &Automation{
		ID: id, Name: "test", Kind: KindCustom, Scopes: []string{"x"},
		TokenHash: "h", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
}

func TestVersionsService_Push(t *testing.T) {
	t.Run("push requires automations:write", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), newFakePerm(nil))
		_, err := s.Push(context.Background(), "someone", "a1", "code", "")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("push 404s on an unknown automation", func(t *testing.T) {
		s := newTestVersionsService(newFakeRepo(), newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.Push(context.Background(), "owner", "missing", "code", "")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("push rejects empty code", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.Push(context.Background(), "owner", "a1", "   ", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("push creates a pending version", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
		v, err := s.Push(context.Background(), "owner", "a1", "console.log(1)", "first push")
		require.NoError(t, err)
		assert.Equal(t, VersionPending, v.Status)
		assert.Equal(t, "owner", v.PusherID)
		assert.Equal(t, 1, v.Sequence)
	})

	t.Run("a second push replaces the first pending, not append", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		vRepo := newFakeVersionsRepo()
		s := newTestVersionsService(autoRepo, vRepo, allowAll("owner"))
		_, err := s.Push(context.Background(), "owner", "a1", "v1", "")
		require.NoError(t, err)
		v2, err := s.Push(context.Background(), "owner", "a1", "v2", "")
		require.NoError(t, err)

		list, err := s.List(context.Background(), "owner", "a1")
		require.NoError(t, err)
		require.Len(t, list, 1, "the replaced pending version is gone, not just superseded")
		assert.Equal(t, v2.ID, list[0].ID)
		assert.Equal(t, "v2", list[0].Code)
	})
}

func TestVersionsService_Activate(t *testing.T) {
	t.Run("merge requires automations:write", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), newFakePerm(nil))
		_, err := s.Activate(context.Background(), "someone", "a1", "v1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("activating the pending version makes it active and the prior active inactive", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		vRepo := newFakeVersionsRepo()
		s := newTestVersionsService(autoRepo, vRepo, allowAll("owner"))

		v1, err := s.Push(context.Background(), "owner", "a1", "v1", "")
		require.NoError(t, err)
		activated, err := s.Activate(context.Background(), "owner", "a1", v1.ID)
		require.NoError(t, err)
		assert.Equal(t, VersionActive, activated.Status)

		v2, err := s.Push(context.Background(), "owner", "a1", "v2", "")
		require.NoError(t, err)
		activated2, err := s.Activate(context.Background(), "owner", "a1", v2.ID)
		require.NoError(t, err)
		assert.Equal(t, VersionActive, activated2.Status)

		old, err := s.Get(context.Background(), "owner", "a1", v1.ID)
		require.NoError(t, err)
		assert.Equal(t, VersionInactive, old.Status, "the previously active version is repointed to inactive, not left active")
	})

	t.Run("rollback re-activates an older inactive version", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		vRepo := newFakeVersionsRepo()
		s := newTestVersionsService(autoRepo, vRepo, allowAll("owner"))

		v1, err := s.Push(context.Background(), "owner", "a1", "v1", "")
		require.NoError(t, err)
		_, err = s.Activate(context.Background(), "owner", "a1", v1.ID)
		require.NoError(t, err)
		v2, err := s.Push(context.Background(), "owner", "a1", "v2", "")
		require.NoError(t, err)
		_, err = s.Activate(context.Background(), "owner", "a1", v2.ID)
		require.NoError(t, err)

		rolledBack, err := s.Activate(context.Background(), "owner", "a1", v1.ID)
		require.NoError(t, err)
		assert.Equal(t, VersionActive, rolledBack.Status)
		assert.Equal(t, v1.ID, rolledBack.ID)
	})

	t.Run("activating an unknown version 404s", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.Activate(context.Background(), "owner", "a1", "missing")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestVersionsService_List(t *testing.T) {
	t.Run("list requires automations:read", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), newFakePerm(nil))
		_, err := s.List(context.Background(), "someone", "a1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
}

func TestVersionsService_Get(t *testing.T) {
	t.Run("get requires automations:read", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), newFakePerm(nil))
		_, err := s.Get(context.Background(), "someone", "a1", "v1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("get 404s on a version that doesn't belong to the automation", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		seedAutomation(t, autoRepo, "a2")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
		v, err := s.Push(context.Background(), "owner", "a1", "v1", "")
		require.NoError(t, err)
		_, err = s.Get(context.Background(), "owner", "a2", v.ID)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestVersionsService_Diff(t *testing.T) {
	t.Run("diff requires automations:read", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), newFakePerm(nil))
		_, _, err := s.Diff(context.Background(), "someone", "a1")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("diff returns nil for both sides on a brand-new automation", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
		active, pending, err := s.Diff(context.Background(), "owner", "a1")
		require.NoError(t, err)
		assert.Nil(t, active)
		assert.Nil(t, pending)
	})

	t.Run("diff returns both sides once there is an active version and a pending push", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
		v1, err := s.Push(context.Background(), "owner", "a1", "v1", "")
		require.NoError(t, err)
		_, err = s.Activate(context.Background(), "owner", "a1", v1.ID)
		require.NoError(t, err)
		v2, err := s.Push(context.Background(), "owner", "a1", "v2", "")
		require.NoError(t, err)

		active, pending, err := s.Diff(context.Background(), "owner", "a1")
		require.NoError(t, err)
		require.NotNil(t, active)
		require.NotNil(t, pending)
		assert.Equal(t, v1.ID, active.ID)
		assert.Equal(t, v2.ID, pending.ID)
	})
}

func TestVersionsService_SeedVersion(t *testing.T) {
	t.Run("the very first seed activates directly, no pending step", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "default-1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))

		v, err := s.SeedVersion(context.Background(), "default-1", "code v1", "seed")
		require.NoError(t, err)
		require.NotNil(t, v)
		assert.Equal(t, VersionActive, v.Status)
		assert.Equal(t, seedPusherID, v.PusherID)
	})

	t.Run("seeding the same code again is a no-op", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "default-1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))

		_, err := s.SeedVersion(context.Background(), "default-1", "code v1", "seed")
		require.NoError(t, err)
		v, err := s.SeedVersion(context.Background(), "default-1", "code v1", "seed")
		require.NoError(t, err)
		assert.Nil(t, v, "unchanged code lands nothing")

		list, err := s.List(context.Background(), "owner", "default-1")
		require.NoError(t, err)
		assert.Len(t, list, 1, "no duplicate version created")
	})

	t.Run("an upgrade with changed code lands as pending, never silently active", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "default-1")
		s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))

		_, err := s.SeedVersion(context.Background(), "default-1", "code v1", "seed")
		require.NoError(t, err)
		v2, err := s.SeedVersion(context.Background(), "default-1", "code v2", "upgrade")
		require.NoError(t, err)
		require.NotNil(t, v2)
		assert.Equal(t, VersionPending, v2.Status)

		activeVersion, pending, err := s.Diff(context.Background(), "owner", "default-1")
		require.NoError(t, err)
		require.NotNil(t, activeVersion)
		require.NotNil(t, pending)
		assert.Equal(t, "code v1", activeVersion.Code, "the upgrade never silently replaces the running active code")
		assert.Equal(t, "code v2", pending.Code)
	})

	t.Run("a real repo failure (not just 'no active yet') is wrapped, not swallowed", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "default-1")
		repo := newErroringVersionsRepo()
		repo.activeErr = errBoom
		s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
		_, err := s.SeedVersion(context.Background(), "default-1", "code", "seed")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("InsertActive failing on the very first seed is wrapped", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "default-1")
		repo := newErroringVersionsRepo()
		repo.insertActiveErr = errBoom
		s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
		_, err := s.SeedVersion(context.Background(), "default-1", "code", "seed")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("ReplacePending failing on an upgrade is wrapped", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "default-1")
		repo := newErroringVersionsRepo()
		s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
		_, err := s.SeedVersion(context.Background(), "default-1", "code v1", "seed")
		require.NoError(t, err)
		repo.replacePendingErr = errBoom
		_, err = s.SeedVersion(context.Background(), "default-1", "code v2", "upgrade")
		require.ErrorIs(t, err, errBoom)
	})
}

func TestVersionsService_MustExist(t *testing.T) {
	t.Run("blank automation id is invalid on every entry point", func(t *testing.T) {
		s := newTestVersionsService(newFakeRepo(), newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.Push(context.Background(), "owner", "  ", "code", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		_, err = s.List(context.Background(), "owner", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		_, err = s.Get(context.Background(), "owner", "", "v1")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		_, err = s.Activate(context.Background(), "owner", "", "v1")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
		_, _, err = s.Diff(context.Background(), "owner", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("list on an unknown automation 404s", func(t *testing.T) {
		s := newTestVersionsService(newFakeRepo(), newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.List(context.Background(), "owner", "missing")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("get on an unknown automation 404s", func(t *testing.T) {
		s := newTestVersionsService(newFakeRepo(), newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.Get(context.Background(), "owner", "missing", "v1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("activate on an unknown automation 404s", func(t *testing.T) {
		s := newTestVersionsService(newFakeRepo(), newFakeVersionsRepo(), allowAll("owner"))
		_, err := s.Activate(context.Background(), "owner", "missing", "v1")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestVersionsService_Activate_BlankVersionID(t *testing.T) {
	autoRepo := newFakeRepo()
	seedAutomation(t, autoRepo, "a1")
	s := newTestVersionsService(autoRepo, newFakeVersionsRepo(), allowAll("owner"))
	_, err := s.Activate(context.Background(), "owner", "a1", "  ")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestVersionsService_Diff_RepoFailures(t *testing.T) {
	t.Run("an active-lookup failure that isn't ErrNotFound is wrapped", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		repo := newErroringVersionsRepo()
		repo.activeErr = errBoom
		s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
		_, _, err := s.Diff(context.Background(), "owner", "a1")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("a pending-lookup failure that isn't ErrNotFound is wrapped", func(t *testing.T) {
		autoRepo := newFakeRepo()
		seedAutomation(t, autoRepo, "a1")
		repo := newErroringVersionsRepo()
		repo.pendingErr = errBoom
		s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
		_, _, err := s.Diff(context.Background(), "owner", "a1")
		require.ErrorIs(t, err, errBoom)
	})
}

func TestVersionsService_List_RepoFailure(t *testing.T) {
	autoRepo := newFakeRepo()
	seedAutomation(t, autoRepo, "a1")
	repo := newErroringVersionsRepo()
	repo.listErr = errBoom
	s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
	_, err := s.List(context.Background(), "owner", "a1")
	require.ErrorIs(t, err, errBoom)
}

func TestVersionsService_Push_RepoFailure(t *testing.T) {
	autoRepo := newFakeRepo()
	seedAutomation(t, autoRepo, "a1")
	repo := newErroringVersionsRepo()
	repo.replacePendingErr = errBoom
	s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
	_, err := s.Push(context.Background(), "owner", "a1", "code", "")
	require.ErrorIs(t, err, errBoom)
}

func TestVersionsService_Activate_RepoFailure(t *testing.T) {
	autoRepo := newFakeRepo()
	seedAutomation(t, autoRepo, "a1")
	repo := newErroringVersionsRepo()
	repo.activateErr = errBoom
	s := newTestVersionsService(autoRepo, repo, allowAll("owner"))
	_, err := s.Activate(context.Background(), "owner", "a1", "v1")
	require.ErrorIs(t, err, errBoom)
}
