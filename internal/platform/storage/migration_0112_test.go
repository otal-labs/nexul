package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration0112_CodesStillOpenAtUpgradeGetAnIdAndCanBeListedAndRevoked: unused codes made before the upgrade carry
// no id, and without one they would never show in the list nor be revocable.
func TestMigration0112_CodesStillOpenAtUpgradeGetAnIdAndCanBeListedAndRevoked(t *testing.T) {
	db := migrateBefore(t, "0112")
	_, err := db.Exec(`
INSERT INTO runner_enrollment_codes (code_hash, name, machine, created_at, expires_at, owner_user_id, computer_id, enrolled_by) VALUES
    ('h-a', 'build-a', 'prod', 1, 4000000000, '', '', 'u-alice'),
    ('h-b', 'build-b', 'prod', 1, 4000000000, '', '', 'u-alice'),
    ('h-c', 'computer-1', '', 1, 4000000000, 'u-alice', 'c-1', '')`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db))
	s := New(db, testEncKey)
	ctx := t.Context()

	codes, err := s.Runners.ListEnrollments(ctx, "u-alice", time.Unix(2, 0))
	require.NoError(t, err)
	require.Len(t, codes, 3, "every old code is listed")
	seen := map[string]bool{}
	for _, c := range codes {
		_, err := uuid.Parse(c.ID)
		require.NoError(t, err, "the id is UUID-shaped, as the service's are")
		seen[c.ID] = true
	}
	assert.Len(t, seen, 3, "each has its own")

	require.NoError(t, s.Runners.RevokeEnrollment(ctx, codes[0].ID))
	codes, err = s.Runners.ListEnrollments(ctx, "u-alice", time.Unix(2, 0))
	require.NoError(t, err)
	assert.Len(t, codes, 2)
}
