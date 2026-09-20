package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/workspace"
)

func TestDeployProjectStore_LinkRepo(t *testing.T) {
	ctx := context.Background()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, s.Projects.Create(ctx, &workspace.Project{ID: "p-other", WorkspaceID: "workspace-default", Name: "Other", Prefix: "OT"}))
	store := deployProjectStore{projects: s.Projects}

	require.NoError(t, store.LinkRepo(ctx, "project-general", "acme", "app"))
	require.NoError(t, store.LinkRepo(ctx, "project-general", "acme", "app"), "re-linking to the same project is a no-op")

	err = store.LinkRepo(ctx, "p-other", "acme", "app")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrs.ErrConflict))
	assert.Contains(t, err.Error(), "already belongs to another project")
}
