package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// TestMigration0091_ExistingRunnersStayDeployRunners upgrades a database at the previous schema: a runner from
// before stays a deploy runner, and no computer, not even an empty id, resolves to it.
func TestMigration0091_ExistingRunnersStayDeployRunners(t *testing.T) {
	db := migrateBefore(t, "0091")
	_, err := db.Exec(`INSERT INTO runners (id, name, last_seen, connected, created_at, version, machine_id) VALUES
    ('r-deploy', 'm1', 1, 0, 1, 'v1', 'machine-1'), ('r-laptop', 'computer-ab12cd34', 1, 0, 1, 'v1', '');`)
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "0091 and every later migration apply on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()

	old, err := s.Runners.GetByID(ctx, "r-deploy")
	require.NoError(t, err)
	assert.False(t, old.Personal())
	_, err = s.Runners.GetByComputer(ctx, "")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "an empty computer id never reaches a deploy runner")

	_, err = db.Exec(`UPDATE runners SET owner_user_id = 'u-alice', computer_id = 'c-laptop' WHERE id = 'r-laptop'`)
	require.NoError(t, err)
	got, err := s.Runners.GetByComputer(ctx, "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, "r-laptop", got.ID)
	assert.Equal(t, "u-alice", got.OwnerUserID)
	assert.True(t, got.Personal())
}
