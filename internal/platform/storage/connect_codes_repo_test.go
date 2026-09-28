package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestConnectCodesRepo(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		setup   func(t *testing.T, s *Store)
		hash    string
		at      time.Time
		want    string
		wantErr error
	}{
		{"unknown hash is not found", func(*testing.T, *Store) {}, "h1", now, "", apperrs.ErrNotFound},
		{"stored hash names its user before expiry", func(t *testing.T, s *Store) {
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h1", now, now.Add(2*time.Minute)))
		}, "h1", now, "u1", nil},
		{"expired hash is not found", func(t *testing.T, s *Store) {
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h1", now, now.Add(2*time.Minute)))
		}, "h1", now.Add(2 * time.Minute), "", apperrs.ErrNotFound},
		{"a newer code replaces the user's earlier one", func(t *testing.T, s *Store) {
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h1", now, now.Add(2*time.Minute)))
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h2", now, now.Add(2*time.Minute)))
		}, "h1", now, "", apperrs.ErrNotFound},
		{"another user's code survives a replace", func(t *testing.T, s *Store) {
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u2", "h2", now, now.Add(2*time.Minute)))
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h1", now, now.Add(2*time.Minute)))
		}, "h2", now, "u2", nil},
		{"consuming is single use", func(t *testing.T, s *Store) {
			require.NoError(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h1", now, now.Add(2*time.Minute)))
			_, err := s.ConnectCodes.ConsumeConnectCode(t.Context(), "h1", now)
			require.NoError(t, err)
		}, "h1", now, "", apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			_, _, err := s.Users.UpsertUser(t.Context(), newTestUser("u1", "42", "onik97"))
			require.NoError(t, err)
			_, _, err = s.Users.UpsertUser(t.Context(), newTestUser("u2", "43", "other"))
			require.NoError(t, err)
			tt.setup(t, s)
			got, err := s.ConnectCodes.ConsumeConnectCode(t.Context(), tt.hash, tt.at)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConnectCodesRepo_ErrorPaths(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	now := time.Now()
	err := s.ConnectCodes.ReplaceConnectCode(t.Context(), "nobody", "h1", now, now.Add(time.Minute))
	require.Error(t, err, "a code needs a user row")

	require.NoError(t, s.Close())
	require.Error(t, s.ConnectCodes.ReplaceConnectCode(t.Context(), "u1", "h1", now, now))
	_, err = s.ConnectCodes.ConsumeConnectCode(t.Context(), "h1", now)
	require.Error(t, err)
}
