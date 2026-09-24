package dns

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func newAccessService(repo *fakeRepo, access *fakeAccessProvider) *Service {
	return NewService(Config{
		Repo:           repo,
		AccessProvider: access,
		EncryptionKey:  testKey(),
		Now:            func() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) },
	})
}

var errZeroTrust = fmt.Errorf("%w: %w", apperrs.ErrInvalid, ErrZeroTrustDisabled)

func TestEnsureAccessServiceToken_Failures(t *testing.T) {
	tests := []struct {
		name      string
		repoErr   error
		accessErr error
		want      error
	}{
		{"repo read fails", errors.New("disk"), nil, nil},
		{"zero trust disabled", nil, errZeroTrust, ErrZeroTrustDisabled},
		{"token rejected", nil, apperrs.ErrUnauthorized, apperrs.ErrFatal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.svcTokenErr = tt.repoErr
			access := newFakeAccessProvider()
			access.err = tt.accessErr
			_, err := newAccessService(repo, access).EnsureAccessServiceToken(t.Context())
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
			assert.Nil(t, repo.svcToken)
		})
	}
}

func TestEnsureAccessServiceToken_UnstorableToken_IsDeletedAtCloudflare(t *testing.T) {
	access := newFakeAccessProvider()
	svc := NewService(Config{Repo: newFakeRepo(), AccessProvider: access})
	_, err := svc.EnsureAccessServiceToken(t.Context())
	require.Error(t, err)
	assert.Equal(t, []string{"st-1"}, access.deleted)
	assert.Empty(t, access.tokens)
}

func TestEnsureAccessServiceToken_UndecryptableSecret_Fails(t *testing.T) {
	repo := newFakeRepo()
	repo.svcToken = &ServiceToken{ID: "st-1", ClientID: "cid", ClientSecret: "not-ciphertext"}
	_, err := newAccessService(repo, newFakeAccessProvider()).EnsureAccessServiceToken(t.Context())
	assert.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestEnsureAccessServiceToken_MintsOncePerInstance(t *testing.T) {
	repo := newFakeRepo()
	access := newFakeAccessProvider()
	svc := newAccessService(repo, access)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			tok, err := svc.EnsureAccessServiceToken(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, "secret-1", tok.ClientSecret)
		})
	}
	wg.Wait()

	assert.Equal(t, 1, access.minted, "concurrent callers share one token")
	require.NotNil(t, repo.svcToken)
	assert.Equal(t, "st-1", repo.svcToken.ID)
	assert.NotEqual(t, "secret-1", repo.svcToken.ClientSecret, "the secret is stored sealed")
}

func TestRotateAccessServiceToken(t *testing.T) {
	t.Run("nothing stored", func(t *testing.T) {
		_, err := newAccessService(newFakeRepo(), newFakeAccessProvider()).RotateAccessServiceToken(t.Context())
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("provider fails, stored secret kept", func(t *testing.T) {
		repo := newFakeRepo()
		access := newFakeAccessProvider()
		svc := newAccessService(repo, access)
		_, err := svc.EnsureAccessServiceToken(t.Context())
		require.NoError(t, err)
		before := repo.svcToken.ClientSecret
		access.err = errors.New("cloudflare down")
		_, err = svc.RotateAccessServiceToken(t.Context())
		require.Error(t, err)
		assert.Equal(t, before, repo.svcToken.ClientSecret)
	})
	t.Run("new secret stored under the same id", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newAccessService(repo, newFakeAccessProvider())
		first, err := svc.EnsureAccessServiceToken(t.Context())
		require.NoError(t, err)
		rotated, err := svc.RotateAccessServiceToken(t.Context())
		require.NoError(t, err)
		assert.Equal(t, first.ID, rotated.ID)
		assert.Equal(t, "rotated-st-1", rotated.ClientSecret)
		assert.Equal(t, first.CreatedAt, rotated.CreatedAt)

		again, err := svc.EnsureAccessServiceToken(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "rotated-st-1", again.ClientSecret, "the rotated secret is what later callers read")
	})
}

func TestDeleteAccessServiceToken(t *testing.T) {
	t.Run("nothing stored is a no-op", func(t *testing.T) {
		access := newFakeAccessProvider()
		require.NoError(t, newAccessService(newFakeRepo(), access).DeleteAccessServiceToken(t.Context()))
		assert.Empty(t, access.deleted)
	})
	t.Run("repo read fails", func(t *testing.T) {
		repo := newFakeRepo()
		repo.svcTokenErr = errors.New("disk")
		assert.Error(t, newAccessService(repo, newFakeAccessProvider()).DeleteAccessServiceToken(t.Context()))
	})
	t.Run("provider fails, row kept", func(t *testing.T) {
		repo := newFakeRepo()
		access := newFakeAccessProvider()
		svc := newAccessService(repo, access)
		_, err := svc.EnsureAccessServiceToken(t.Context())
		require.NoError(t, err)
		access.err = errors.New("cloudflare down")
		require.Error(t, svc.DeleteAccessServiceToken(t.Context()))
		assert.NotNil(t, repo.svcToken)
	})
	t.Run("removed at cloudflare and locally", func(t *testing.T) {
		repo := newFakeRepo()
		access := newFakeAccessProvider()
		svc := newAccessService(repo, access)
		_, err := svc.EnsureAccessServiceToken(t.Context())
		require.NoError(t, err)
		require.NoError(t, svc.DeleteAccessServiceToken(t.Context()))
		assert.Nil(t, repo.svcToken)
		assert.Empty(t, access.tokens)
	})
}

func TestCreateAccessApp(t *testing.T) {
	t.Run("blank hostname", func(t *testing.T) {
		_, err := newAccessService(newFakeRepo(), newFakeAccessProvider()).CreateAccessApp(t.Context(), "  ")
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("zero trust disabled", func(t *testing.T) {
		access := newFakeAccessProvider()
		access.err = errZeroTrust
		_, err := newAccessService(newFakeRepo(), access).CreateAccessApp(t.Context(), "laptop.example.com")
		assert.ErrorIs(t, err, ErrZeroTrustDisabled)
	})
	t.Run("app admits the instance token", func(t *testing.T) {
		access := newFakeAccessProvider()
		svc := newAccessService(newFakeRepo(), access)
		one, err := svc.CreateAccessApp(t.Context(), "laptop.example.com")
		require.NoError(t, err)
		two, err := svc.CreateAccessApp(t.Context(), "desktop.example.com")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{one: "st-1", two: "st-1"}, access.apps)
		assert.Equal(t, 1, access.minted)
	})
}

func TestCreateAccessApp_ProviderFails(t *testing.T) {
	repo := newFakeRepo()
	access := newFakeAccessProvider()
	svc := newAccessService(repo, access)
	_, err := svc.EnsureAccessServiceToken(t.Context())
	require.NoError(t, err)
	access.err = apperrs.ErrConflict
	_, err = svc.CreateAccessApp(t.Context(), "laptop.example.com")
	assert.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestDeleteAccessApp(t *testing.T) {
	access := newFakeAccessProvider()
	svc := newAccessService(newFakeRepo(), access)
	assert.ErrorIs(t, svc.DeleteAccessApp(t.Context(), ""), apperrs.ErrInvalid)

	id, err := svc.CreateAccessApp(t.Context(), "laptop.example.com")
	require.NoError(t, err)
	access.err = errors.New("cloudflare down")
	require.Error(t, svc.DeleteAccessApp(t.Context(), id))
	access.err = nil
	require.NoError(t, svc.DeleteAccessApp(t.Context(), id))
	assert.Empty(t, access.apps)
}

func TestAccessProviderFor_Wiring(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want error
	}{
		{"no token source", Config{Repo: newFakeRepo()}, apperrs.ErrFatal},
		{"no constructor", Config{Repo: newFakeRepo(), Tokens: &fakeTokenProvider{token: "at"}}, apperrs.ErrInvalid},
		{"constructor fails", Config{Repo: newFakeRepo(), Tokens: &fakeTokenProvider{token: "at"},
			NewAccessProvider: func(context.Context, string) (AccessProvider, error) { return nil, errors.New("boom") }}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewService(tt.cfg).DeleteAccessApp(t.Context(), "app-1")
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}

	built := newFakeAccessProvider()
	svc := NewService(Config{Repo: newFakeRepo(), Tokens: &fakeTokenProvider{token: "at"}, EncryptionKey: testKey(),
		NewAccessProvider: func(_ context.Context, token string) (AccessProvider, error) {
			assert.Equal(t, "at", token)
			return built, nil
		}})
	_, err := svc.CreateAccessApp(t.Context(), "laptop.example.com")
	require.NoError(t, err)
	assert.Len(t, built.apps, 1)
}
