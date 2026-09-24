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

func TestDeployProjectStore_IsTestsRepo(t *testing.T) {
	ctx := t.Context()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	store := deployProjectStore{projects: s.Projects}
	require.NoError(t, store.LinkRepo(ctx, "project-general", "acme", "app"))
	require.NoError(t, s.Projects.AddRepo(ctx, "project-general",
		workspace.RepoRef{Owner: "acme", Name: "e2e", FullName: "acme/e2e", ConnectorID: "github", Role: workspace.RepoRoleTests}))

	tests, err := store.IsTestsRepo(ctx, "acme", "e2e")
	require.NoError(t, err)
	assert.True(t, tests)
	tests, err = store.IsTestsRepo(ctx, "acme", "app")
	require.NoError(t, err)
	assert.False(t, tests, "a linked repository is an app repository")
	tests, err = store.IsTestsRepo(ctx, "acme", "unlinked")
	require.NoError(t, err)
	assert.False(t, tests)
}
