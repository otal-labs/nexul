package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/imports"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/roles"
	"github.com/otal-labs/nexul/internal/tenancy"
)

func newStoredImport(t *testing.T, s *Store, id, credential string) *imports.Import {
	t.Helper()
	now := time.Unix(1_800_000_000, 0).UTC()
	imp := &imports.Import{
		ID: id, WorkspaceID: "workspace-default", ProjectID: "project-general", Source: "github", State: imports.StateScanning,
		CreatedBy: "u-owner", Credential: credential, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, s.Imports.Create(t.Context(), imp))
	return imp
}

func storedCredential(t *testing.T, s *Store, id string) sql.NullString {
	t.Helper()
	var c sql.NullString
	require.NoError(t, s.db.QueryRowContext(t.Context(), `SELECT credential_encrypted FROM imports WHERE id = ?`, id).Scan(&c))
	return c
}

func TestImportsRepo_SealsTheCredentialAndClearsItInEveryFinalState(t *testing.T) {
	t.Parallel()
	for _, final := range []imports.State{imports.StateCompleted, imports.StateFailed, imports.StateCancelled} {
		t.Run(string(final), func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			imp := newStoredImport(t, s, "imp-1", "lin_api_secret")
			sealed := storedCredential(t, s, imp.ID)
			require.True(t, sealed.Valid)
			assert.NotContains(t, sealed.String, "lin_api_secret", "sealed at rest")
			got, err := s.Imports.Get(t.Context(), imp.ID)
			require.NoError(t, err)
			assert.Equal(t, "lin_api_secret", got.Credential)

			got.State = final
			ok, err := s.Imports.Update(t.Context(), got, imports.StateScanning)
			require.NoError(t, err)
			require.True(t, ok)
			assert.False(t, storedCredential(t, s, imp.ID).Valid)
			got, err = s.Imports.Get(t.Context(), imp.ID)
			require.NoError(t, err)
			assert.Empty(t, got.Credential)
		})
	}
}

func TestImportsRepo_TheTableRefusesAFinalStateThatKeepsTheCredential(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	newStoredImport(t, s, "imp-1", "secret")
	_, err := s.db.ExecContext(t.Context(), `UPDATE imports SET state = 'completed' WHERE id = 'imp-1'`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CHECK")
}

func TestImportsRepo_OneOpenImportPerWorkspace_AndUpdateComparesTheState(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	imp := newStoredImport(t, s, "imp-1", "")
	err := s.Imports.Create(t.Context(), &imports.Import{ID: "imp-2", WorkspaceID: "workspace-default", ProjectID: "project-general", State: imports.StateScanning})
	require.ErrorIs(t, err, apperrs.ErrConflict)

	imp.State = imports.StateImporting
	ok, err := s.Imports.Update(t.Context(), imp, imports.StateReady)
	require.NoError(t, err)
	assert.False(t, ok, "a stale state writes nothing")

	imp.State, imp.Done, imp.Report.Cursor = imports.StateReady, 3, "4"
	ok, err = s.Imports.Update(t.Context(), imp, imports.StateScanning)
	require.NoError(t, err)
	require.True(t, ok)
	open, err := s.Imports.Open(t.Context())
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, 3, open[0].Done)
	assert.Equal(t, "4", open[0].Report.Cursor)

	imp.State = imports.StateCancelled
	_, err = s.Imports.Update(t.Context(), imp, imports.StateReady)
	require.NoError(t, err)
	newStoredImport(t, s, "imp-3", "")
	list, err := s.Imports.ListByWorkspace(t.Context(), "workspace-default", 10)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestImportsRepo_RecordsItemsAndTheSourceIDsTheyCameFrom(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	newStoredImport(t, s, "imp-1", "")
	ctx := t.Context()
	_, ok, err := s.Imports.Item(ctx, "imp-1", "ticket", "77")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, s.Imports.Record(ctx, "workspace-default", "github", imports.Item{ImportID: "imp-1", Kind: "ticket", SourceID: "78", State: imports.ItemFailed, Error: "refused"}))
	_, before, err := s.Imports.Imported(ctx, "workspace-default", "github", "ticket", "78")
	require.NoError(t, err)
	assert.False(t, before, "a failed record is tried again by a later import")

	require.NoError(t, s.Imports.Record(ctx, "workspace-default", "github", imports.Item{ImportID: "imp-1", Kind: "ticket", SourceID: "77", NexulID: "t-1", State: imports.ItemCreated}))
	require.NoError(t, s.Imports.Record(ctx, "workspace-default", "github", imports.Item{ImportID: "imp-1", Kind: "ticket", SourceID: "77", NexulID: "t-1", State: imports.ItemFilled}))
	item, ok, err := s.Imports.Item(ctx, "imp-1", "ticket", "77")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, imports.ItemFilled, item.State)
	id, before, err := s.Imports.Imported(ctx, "workspace-default", "github", "ticket", "77")
	require.NoError(t, err)
	assert.True(t, before)
	assert.Equal(t, "t-1", id)
}

func TestImportsRepo_PeopleGroupsEachMembersGitHubLoginsAndEmails(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, s.Roles.Create(ctx, &roles.Role{ID: "role-member", WorkspaceID: "workspace-default", Name: "Member", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	for _, id := range []auth.Identity{
		{UserID: "u-a", Provider: auth.ProviderGitHub, ProviderUserID: "1", Login: "Alice"},
		{UserID: "u-b", Provider: "google", ProviderUserID: "2", Login: "Bob@Example.com"},
	} {
		_, _, err := s.Users.UpsertUser(ctx, &id)
		require.NoError(t, err)
		require.NoError(t, s.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: id.UserID, WorkspaceID: "workspace-default", RoleID: "role-member", CreatedAt: time.Now()}))
	}
	people, err := s.Imports.People(ctx, "workspace-default")
	require.NoError(t, err)
	assert.ElementsMatch(t, []imports.Member{
		{UserID: "u-a", Login: "Alice", GitHubLogins: []string{"alice"}},
		{UserID: "u-b", Login: "Bob@Example.com", Emails: []string{"bob@example.com"}},
	}, people)
}

func TestImportsRepo_MoveChangesTheStateAloneAndKeepsTheLatestCounts(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	stale := newStoredImport(t, s, "imp-1", "secret")
	saved := *stale
	saved.Done, saved.Report.Cursor = 40, "3"
	ok, err := s.Imports.Update(t.Context(), &saved, imports.StateScanning)
	require.NoError(t, err)
	require.True(t, ok)

	at := time.Unix(1_800_000_100, 0).UTC()
	var announced *imports.Import
	moved, ok, err := s.Imports.Move(t.Context(), stale.ID, imports.StateScanning, imports.StateCancelled, "stopped by its Owner", at,
		func(imp *imports.Import) eventbus.OutboxEvent {
			announced = imp
			return eventbus.OutboxEvent{ID: "evt-1", Topic: imports.TopicProgress, Payload: imports.ProgressEvent{Import: *imp}}
		})
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, 40, moved.Done, "the loop's newer count survives the cancel")
	assert.Equal(t, "3", moved.Report.Cursor)
	assert.Equal(t, "stopped by its Owner", moved.Report.Error)
	assert.Equal(t, imports.StateCancelled, moved.State)
	assert.Equal(t, at, moved.UpdatedAt)
	assert.Same(t, moved, announced)
	assert.False(t, storedCredential(t, s, stale.ID).Valid)

	_, ok, err = s.Imports.Move(t.Context(), stale.ID, imports.StateScanning, imports.StateFailed, "", at, func(*imports.Import) eventbus.OutboxEvent {
		t.Fatal("no event for a move that did not happen")
		return eventbus.OutboxEvent{}
	})
	require.NoError(t, err)
	assert.False(t, ok, "a stale state moves nothing")
}
