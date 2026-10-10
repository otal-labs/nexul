package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/runner"
)

// TestMigration0093_ACodeKeepsItsPersonAndComputerAndAComputerHasOneRunner upgrades a database holding a deploy
// runner's code: it stays a deploy code, a personal code round-trips its person and computer, and a second runner for
// one computer is refused while deploy runners, which name no computer, are not.
func TestMigration0093_ACodeKeepsItsPersonAndComputerAndAComputerHasOneRunner(t *testing.T) {
	db := migrateBefore(t, "0093")
	_, err := db.Exec(`INSERT INTO runner_enrollment_codes (code_hash, name, machine, created_at, expires_at) VALUES ('h-deploy', 'edge', '', 1, 4102444800)`)
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()
	now := time.Unix(100, 0).UTC()

	old, err := s.Runners.GetEnrollment(ctx, "h-deploy", now)
	require.NoError(t, err)
	assert.False(t, old.Personal())
	require.NoError(t, s.Runners.CreateEnrollment(ctx, &runner.EnrollmentCode{CodeHash: "h-laptop", Name: "computer-ab12cd34", OwnerUserID: "u-alice", ComputerID: "c-laptop", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}))
	personal, err := s.Runners.GetEnrollment(ctx, "h-laptop", now)
	require.NoError(t, err)
	assert.Equal(t, "u-alice", personal.OwnerUserID)
	assert.Equal(t, "c-laptop", personal.ComputerID)

	for _, r := range []*runner.Runner{{ID: "r-1", Name: "edge-1"}, {ID: "r-2", Name: "edge-2"}, {ID: "r-3", Name: "computer-1", OwnerUserID: "u-alice", ComputerID: "c-laptop"}} {
		require.NoError(t, s.Runners.Create(ctx, r))
	}
	err = s.Runners.Create(ctx, &runner.Runner{ID: "r-4", Name: "computer-2", OwnerUserID: "u-alice", ComputerID: "c-laptop"})
	require.ErrorIs(t, err, apperrs.ErrConflict)
}
