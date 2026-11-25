package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestMigration0169_AGrantFromBeforeRotatesAndThenKnowsItsPreviousRefresh upgrades a grant minted before 0169.
func TestMigration0169_AGrantFromBeforeRotatesAndThenKnowsItsPreviousRefresh(t *testing.T) {
	db := migrateBefore(t, "0169")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1);
INSERT INTO personal_access_tokens (id, user_id, name, token_hash, prefix, created_at, oauth_client_id, refresh_hash, expires_at)
VALUES ('g1', 'u-alice', 'Acme', 'access-1', 'cess-1', 100, 'client-1', 'refresh-1', 5000);`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0169 and every later migration apply on top, as an upgrade would")
	s := New(db, testEncKey)
	ctx := t.Context()

	_, err = s.PATs.GetByPreviousRefresh(ctx, "refresh-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	later := time.Unix(9000, 0).UTC()
	_, err = s.PATs.RotateByRefresh(ctx, "refresh-1", "client-1", time.Unix(0, 0),
		auth.PersonalAccessToken{TokenHash: "access-2", RefreshHash: "refresh-2", Prefix: "cess-2", ExpiresAt: &later})
	require.NoError(t, err)
	got, err := s.PATs.GetByPreviousRefresh(ctx, "refresh-1")
	require.NoError(t, err)
	assert.Equal(t, "g1", got.ID)
}
