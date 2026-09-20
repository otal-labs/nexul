package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/connectors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func openConnectorsStore(t *testing.T) *Store {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return New(db, testEncKey)
}

// TestConnectorsRepo_GetCredentials_NeverConnected proves a connector that
// was never connected returns a clean apperrs.ErrNotFound, not a zero value.
func TestConnectorsRepo_GetCredentials_NeverConnected(t *testing.T) {
	t.Parallel()
	s := openConnectorsStore(t)
	ctx := context.Background()

	_, err := s.Connectors.GetCredentials(ctx, "github")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "never-connected connector must be ErrNotFound")
}

// TestConnectorsRepo_SaveGetDelete_RoundTrip proves upsert + get round-trip
// correctly, that the raw DB columns hold ciphertext (not the plaintext
// tokens), that reconnecting overwrites rather than erroring, and that
// delete actually removes the row (subsequent get is not-found again).
func TestConnectorsRepo_SaveGetDelete_RoundTrip(t *testing.T) {
	t.Parallel()
	s := openConnectorsStore(t)
	ctx := context.Background()

	c := connectors.Credentials{
		ConnectorID:  "github",
		AccessToken:  "gho_first-access-token",
		RefreshToken: "ghr_first-refresh-token",
		ExpiresAt:    time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC),
		ConnectedBy:  "user-1",
		ConnectedAt:  time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, s.Connectors.SaveCredentials(ctx, c))

	got, err := s.Connectors.GetCredentials(ctx, "github")
	require.NoError(t, err)
	assert.Equal(t, c.ConnectorID, got.ConnectorID)
	assert.Equal(t, c.AccessToken, got.AccessToken)
	assert.Equal(t, c.RefreshToken, got.RefreshToken)
	assert.Equal(t, c.ExpiresAt, got.ExpiresAt)
	assert.Equal(t, c.ConnectedBy, got.ConnectedBy)
	assert.Equal(t, c.ConnectedAt, got.ConnectedAt)

	status := got.Status()
	assert.True(t, status.Configured)
	assert.Equal(t, c.ConnectedBy, status.ConnectedBy)

	// Verify the tokens are genuinely encrypted at rest: the raw columns
	// must not contain the plaintext.
	var rawAccess, rawRefresh string
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT access_token, refresh_token FROM connector_credentials WHERE connector_id = ?`, "github").
		Scan(&rawAccess, &rawRefresh))
	assert.NotEqual(t, c.AccessToken, rawAccess, "access token must not be stored as plaintext")
	assert.NotEqual(t, c.RefreshToken, rawRefresh, "refresh token must not be stored as plaintext")
	assert.NotEmpty(t, rawAccess)
	assert.NotEmpty(t, rawRefresh)

	// Reconnecting (same connector, new tokens) overwrites rather than
	// erroring or leaving a second row.
	c2 := connectors.Credentials{
		ConnectorID:  "github",
		AccessToken:  "gho_second-access-token",
		RefreshToken: "ghr_second-refresh-token",
		ExpiresAt:    time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC),
		ConnectedBy:  "user-2",
		ConnectedAt:  time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, s.Connectors.SaveCredentials(ctx, c2))

	got2, err := s.Connectors.GetCredentials(ctx, "github")
	require.NoError(t, err)
	assert.Equal(t, c2.AccessToken, got2.AccessToken)
	assert.Equal(t, c2.RefreshToken, got2.RefreshToken)
	assert.Equal(t, c2.ConnectedBy, got2.ConnectedBy)

	var count int
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM connector_credentials WHERE connector_id = ?`, "github").Scan(&count))
	assert.Equal(t, 1, count, "reconnect must overwrite the single row, not add a second")

	// Delete removes the row; a subsequent get is not-found again.
	require.NoError(t, s.Connectors.DeleteCredentials(ctx, "github"))
	_, err = s.Connectors.GetCredentials(ctx, "github")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound), "deleted connector must be ErrNotFound again")

	// Deleting an already-disconnected connector is a no-op, not an error.
	require.NoError(t, s.Connectors.DeleteCredentials(ctx, "github"))
}

// TestConnectorsRepo_SaveCredentials_NoRefreshToken proves an empty refresh
// token round-trips as empty rather than round-tripping through Encrypt (a
// connector may only ever hand back an access token).
func TestConnectorsRepo_SaveCredentials_NoRefreshToken(t *testing.T) {
	t.Parallel()
	s := openConnectorsStore(t)
	ctx := context.Background()

	c := connectors.Credentials{
		ConnectorID: "tally",
		AccessToken: "tally-token",
		ConnectedBy: "user-1",
		ConnectedAt: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, s.Connectors.SaveCredentials(ctx, c))

	got, err := s.Connectors.GetCredentials(ctx, "tally")
	require.NoError(t, err)
	assert.Equal(t, "", got.RefreshToken)

	var rawRefresh string
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT refresh_token FROM connector_credentials WHERE connector_id = ?`, "tally").Scan(&rawRefresh))
	assert.Equal(t, "", rawRefresh)
}

// TestConnectorsRepo_ManualFields_RoundTripEncrypted proves ticket 10's
// manual-credential storage: field values round-trip through Save/Get, the
// raw manual_fields column is ciphertext (not plaintext JSON), and there's
// no access/refresh token needed for the credential to read back as
// configured.
func TestConnectorsRepo_ManualFields_RoundTripEncrypted(t *testing.T) {
	t.Parallel()
	s := openConnectorsStore(t)
	ctx := context.Background()

	c := connectors.Credentials{
		ConnectorID: "livekit",
		ManualFields: map[string]string{
			"ws_url":     "wss://lk.example.com",
			"api_key":    "APIabc123",
			"api_secret": "supersecret",
		},
		ConnectedBy: "user-1",
		ConnectedAt: time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, s.Connectors.SaveCredentials(ctx, c))

	got, err := s.Connectors.GetCredentials(ctx, "livekit")
	require.NoError(t, err)
	assert.Equal(t, c.ManualFields, got.ManualFields)
	assert.Equal(t, "", got.AccessToken)
	assert.True(t, got.Status().Configured, "manual credential must read back configured even with no access token")

	var rawManual string
	require.NoError(t, s.db.QueryRowContext(ctx,
		`SELECT manual_fields FROM connector_credentials WHERE connector_id = ?`, "livekit").Scan(&rawManual))
	assert.NotContains(t, rawManual, "supersecret", "manual fields must not be stored as plaintext")
	assert.NotEmpty(t, rawManual)

	require.NoError(t, s.Connectors.DeleteCredentials(ctx, "livekit"))
	_, err = s.Connectors.GetCredentials(ctx, "livekit")
	assert.True(t, errors.Is(err, apperrs.ErrNotFound))
}
