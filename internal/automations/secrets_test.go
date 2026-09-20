package automations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestSecretsService(repo SecretsRepo, perm PermissionGate) *SecretsService {
	s := NewSecretsService(repo, perm)
	s.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestSecretsService_Set(t *testing.T) {
	t.Run("set requires automations:write", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), newFakePerm(nil))
		err := s.Set(context.Background(), "someone", "API_KEY", "shh")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("rejects a name that isn't a valid identifier", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		for _, bad := range []string{"", "1LEADING_DIGIT", "has space", "has-dash", "has.dot"} {
			err := s.Set(context.Background(), "owner", bad, "value")
			require.ErrorIsf(t, err, apperrs.ErrInvalid, "name %q", bad)
		}
	})

	t.Run("rejects an empty value", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		err := s.Set(context.Background(), "owner", "API_KEY", "")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("set stores the value and lists the name", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		require.NoError(t, s.Set(context.Background(), "owner", "API_KEY", "sk-123"))

		list, err := s.List(context.Background(), "owner")
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, "API_KEY", list[0].Name)

		all, err := s.Secrets(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "sk-123", all["API_KEY"])
	})

	t.Run("set replaces an existing value wholesale", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		require.NoError(t, s.Set(context.Background(), "owner", "API_KEY", "old"))
		require.NoError(t, s.Set(context.Background(), "owner", "API_KEY", "new"))

		all, err := s.Secrets(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "new", all["API_KEY"])
		list, err := s.List(context.Background(), "owner")
		require.NoError(t, err)
		assert.Len(t, list, 1, "replace, not append")
	})
}

func TestSecretsService_Delete(t *testing.T) {
	t.Run("delete requires automations:write", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), newFakePerm(nil))
		err := s.Delete(context.Background(), "someone", "API_KEY")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("deleting a name that was never set is a no-op", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		err := s.Delete(context.Background(), "owner", "NEVER_SET")
		require.NoError(t, err)
	})

	t.Run("delete removes the secret", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		require.NoError(t, s.Set(context.Background(), "owner", "API_KEY", "sk-123"))
		require.NoError(t, s.Delete(context.Background(), "owner", "API_KEY"))

		list, err := s.List(context.Background(), "owner")
		require.NoError(t, err)
		assert.Empty(t, list)
	})
}

func TestSecretsService_List(t *testing.T) {
	t.Run("list requires automations:read", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), newFakePerm(nil))
		_, err := s.List(context.Background(), "someone")
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})

	t.Run("list never carries a value field", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		require.NoError(t, s.Set(context.Background(), "owner", "API_KEY", "sk-123"))
		list, err := s.List(context.Background(), "owner")
		require.NoError(t, err)
		require.Len(t, list, 1)
		// SecretMeta has no Value field at all — this asserts the shape,
		// not a nil check: a Value field would have to be added to break
		// this, which is the point (see secrets_handler_test.go for the
		// same guarantee over the wire).
		assert.Equal(t, SecretMeta{Name: "API_KEY", CreatedAt: list[0].CreatedAt, UpdatedAt: list[0].UpdatedAt}, list[0])
	})
}

func TestSecretsService_Secrets(t *testing.T) {
	t.Run("Secrets is not permission-gated (internal delivery-layer seam)", func(t *testing.T) {
		repo := newFakeSecretsRepo()
		require.NoError(t, repo.Set(context.Background(), "API_KEY", "sk-123", time.Now()))
		s := newTestSecretsService(repo, newFakePerm(nil))
		all, err := s.Secrets(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "sk-123", all["API_KEY"])
	})

	t.Run("a repo failure is wrapped, not swallowed", func(t *testing.T) {
		repo := newErroringSecretsRepo()
		repo.allErr = errBoom
		s := newTestSecretsService(repo, newFakePerm(nil))
		_, err := s.Secrets(context.Background())
		require.ErrorIs(t, err, errBoom)
	})
}

func TestSecretsService_RepoFailures(t *testing.T) {
	t.Run("Set wraps a repo failure", func(t *testing.T) {
		repo := newErroringSecretsRepo()
		repo.setErr = errBoom
		s := newTestSecretsService(repo, allowAll("owner"))
		err := s.Set(context.Background(), "owner", "API_KEY", "value")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("Delete wraps a repo failure", func(t *testing.T) {
		repo := newErroringSecretsRepo()
		repo.deleteErr = errBoom
		s := newTestSecretsService(repo, allowAll("owner"))
		err := s.Delete(context.Background(), "owner", "API_KEY")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("Delete rejects a blank name", func(t *testing.T) {
		s := newTestSecretsService(newFakeSecretsRepo(), allowAll("owner"))
		err := s.Delete(context.Background(), "owner", "   ")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("List wraps a repo failure", func(t *testing.T) {
		repo := newErroringSecretsRepo()
		repo.listErr = errBoom
		s := newTestSecretsService(repo, allowAll("owner"))
		_, err := s.List(context.Background(), "owner")
		require.ErrorIs(t, err, errBoom)
	})
}
