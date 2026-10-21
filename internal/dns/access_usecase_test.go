package dns

import (
	"context"
	"errors"
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
		repo, access := storedServiceToken()
		svc := newAccessService(repo, access)
		access.err = errors.New("cloudflare down")
		require.Error(t, svc.DeleteAccessServiceToken(t.Context()))
		assert.NotNil(t, repo.svcToken)
	})
	t.Run("removed at cloudflare and locally", func(t *testing.T) {
		repo, access := storedServiceToken()
		svc := newAccessService(repo, access)
		require.NoError(t, svc.DeleteAccessServiceToken(t.Context()))
		assert.Nil(t, repo.svcToken)
		assert.Empty(t, access.tokens)
	})
}

func TestDeleteAccessApp(t *testing.T) {
	access := newFakeAccessProvider()
	svc := newAccessService(newFakeRepo(), access)
	assert.ErrorIs(t, svc.DeleteAccessApp(t.Context(), ""), apperrs.ErrInvalid)

	id := "app-laptop.example.com"
	access.apps[id] = "st-1"
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
			NewAccessProvider: func(context.Context, string, string) (AccessProvider, error) { return nil, errors.New("boom") }}, nil},
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
		NewAccessProvider: func(_ context.Context, token, _ string) (AccessProvider, error) {
			assert.Equal(t, "at", token)
			return built, nil
		}})
	built.apps["app-1"] = "st-1"
	require.NoError(t, svc.DeleteAccessApp(t.Context(), "app-1"))
	assert.Empty(t, built.apps)
}

// storedServiceToken is an instance whose token pairing through a tunnel once minted, stored and at Cloudflare.
func storedServiceToken() (*fakeRepo, *fakeAccessProvider) {
	repo := newFakeRepo()
	repo.svcToken = &ServiceToken{ID: "st-1", ClientID: "cid-st-1", ClientSecret: "sealed"}
	access := newFakeAccessProvider()
	access.tokens["st-1"] = "secret-1"
	return repo, access
}
