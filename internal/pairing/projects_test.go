package pairing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newTestServiceWithLister(repo *fakeRepo, lister func(context.Context, harness.Session) ([]harness.Project, error)) *Service {
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
	exch.ListProjectsFn = lister
	return NewService(Config{
		Repo:          repo,
		Harnesses:     registry(exch),
		EncryptionKey: testEncKey,
		Now:           func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) },
	})
}

func TestListProjects(t *testing.T) {
	t.Parallel()

	t.Run("returns the lister's registry with a decrypted session", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		var gotToken string
		svc := newTestServiceWithLister(repo, func(_ context.Context, c harness.Session) ([]harness.Project, error) {
			gotToken = c.BearerToken
			return []harness.Project{{ID: "p1", Title: "App"}}, nil
		})
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		projects, err := svc.ListProjects(context.Background(), "u1", c.ID)
		require.NoError(t, err)
		assert.Equal(t, []harness.Project{{ID: "p1", Title: "App"}}, projects)
		assert.Equal(t, "b", gotToken, "the lister must receive the decrypted bearer")
	})

	t.Run("someone else's computer is not found", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestServiceWithLister(repo, func(context.Context, harness.Session) ([]harness.Project, error) { return nil, nil })
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		_, err = svc.ListProjects(context.Background(), "u2", c.ID)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})

	t.Run("unregistered kind is fatal", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}, version: "0.0.34"})
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		stored := repo.computers[c.ID]
		stored.Kind = "opencode"
		repo.computers[c.ID] = stored
		_, err = svc.ListProjects(context.Background(), "u1", c.ID)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}

func TestListProviders(t *testing.T) {
	t.Parallel()

	t.Run("returns the lister's providers with a decrypted session", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		var gotToken string
		exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: 30 * 24 * time.Hour}, version: "0.0.34"}
		exch.ListProvidersFn = func(_ context.Context, s harness.Session) ([]harness.Provider, error) {
			gotToken = s.BearerToken
			return []harness.Provider{{ID: "claude", Name: "Claude Code", Models: []harness.ProviderModel{{Slug: "claude-sonnet-4-5", Name: "Sonnet 4.5", IsDefault: true}}}}, nil
		}
		svc := newTestService(repo, exch)
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		providers, err := svc.ListProviders(context.Background(), "u1", c.ID)
		require.NoError(t, err)
		require.Len(t, providers, 1)
		assert.Equal(t, "claude", providers[0].ID)
		assert.Equal(t, "b", gotToken, "the lister must receive the decrypted bearer")
	})

	t.Run("unregistered kind is fatal", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		svc := newTestService(repo, &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}, version: "0.0.34"})
		c, err := svc.Pair(context.Background(), "u1", harness.KindT3Code, "Home", "https://h.example.com", "tok")
		require.NoError(t, err)
		stored := repo.computers[c.ID]
		stored.Kind = "opencode"
		repo.computers[c.ID] = stored
		_, err = svc.ListProviders(context.Background(), "u1", c.ID)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
}
