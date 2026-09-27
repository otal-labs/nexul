package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupCodesRepo(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		setup func(t *testing.T, s *Store)
		hash  string
		at    time.Time
		want  bool
	}{
		{"unknown hash is invalid", func(*testing.T, *Store) {}, "h1", now, false},
		{"stored hash is valid before expiry", func(t *testing.T, s *Store) {
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now.Add(time.Hour)))
		}, "h1", now, true},
		{"expired hash is invalid", func(t *testing.T, s *Store) {
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now.Add(time.Hour)))
		}, "h1", now.Add(time.Hour), false},
		{"replacing drops the earlier code", func(t *testing.T, s *Store) {
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now.Add(time.Hour)))
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h2", now, now.Add(time.Hour)))
		}, "h1", now, false},
		{"replacing with the same hash keeps it", func(t *testing.T, s *Store) {
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now.Add(time.Hour)))
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now.Add(time.Hour)))
		}, "h1", now, true},
		{"clearing leaves nothing valid", func(t *testing.T, s *Store) {
			require.NoError(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now.Add(time.Hour)))
			require.NoError(t, s.SetupCodes.ClearSetupCodes(t.Context()))
		}, "h1", now, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			tt.setup(t, s)
			got, err := s.SetupCodes.SetupCodeValid(t.Context(), tt.hash, tt.at)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSetupCodesRepo_ClosedDB_Errors(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Close())
	now := time.Now()
	require.Error(t, s.SetupCodes.ReplaceSetupCode(t.Context(), "h1", now, now))
	_, err := s.SetupCodes.SetupCodeValid(t.Context(), "h1", now)
	require.Error(t, err)
	require.Error(t, s.SetupCodes.ClearSetupCodes(t.Context()))
}
