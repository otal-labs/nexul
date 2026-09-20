package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
)

func TestReadFunctions_ClosedDB_Error(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.db.Close())

	_, err := s.Docs.GetByID(context.Background(), "x")
	require.Error(t, err)
	_, err = s.Docs.List(context.Background())
	require.Error(t, err)
	_, err = s.Docs.Search(context.Background(), "x", 10)
	require.Error(t, err)

	_, err = s.Tickets.GetByID(context.Background(), "x")
	require.Error(t, err)
	_, err = s.Tickets.List(context.Background())
	require.Error(t, err)
	_, err = s.Tickets.ListByDoc(context.Background(), "x")
	require.Error(t, err)
	_, err = s.Tickets.Search(context.Background(), "x", 10)
	require.Error(t, err)

	_, err = s.Deploys.GetByID(context.Background(), "x")
	require.Error(t, err)
	_, err = s.Deploys.List(context.Background())
	require.Error(t, err)
	_, err = s.Deploys.ListByService(context.Background(), "api")
	require.Error(t, err)
	_, err = s.Deploys.ListByStatus(context.Background(), deploy.StatusPending)
	require.Error(t, err)

	_, err = s.Topology.Get(context.Background(), "prod")
	require.Error(t, err)

	_, err = s.Runners.GetByID(context.Background(), "x")
	require.Error(t, err)
	_, err = s.Runners.List(context.Background())
	require.Error(t, err)

	_, err = s.Outbox.Unpublished(context.Background(), 10)
	require.Error(t, err)
	_, err = s.ProcessedEvents.Seen(context.Background(), "x")
	require.Error(t, err)

	_, err = s.DeadLetters.Get(context.Background(), "x")
	require.Error(t, err)
	_, err = s.DeadLetters.List(context.Background(), 10, 0)
	require.Error(t, err)
}

func TestMigrationApplied_ClosedDB_Error(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, db.Close())

	_, err = migrationApplied(db, "0001_schema")
	require.Error(t, err)
}

func TestApplyMigration_ClosedDB_Error(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, db.Close())

	err = applyMigration(db, "bad", "SELECT 1;")
	require.Error(t, err)
}
