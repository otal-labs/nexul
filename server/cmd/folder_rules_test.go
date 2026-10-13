package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/docs"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/testutil"
	"github.com/otal-labs/nexul/internal/workspace"
)

// TestFolderRules_MovingADocChangesWhatReachesIt: a folder's Deny follows a doc in and out within one memo, on every read.
func TestFolderRules_MovingADocChangesWhatReachesIt(t *testing.T) {
	f := newPermFixture(t)
	s := f.svc
	folder, err := s.docsSvc.CreateFolder(as(uOwner), pGeneral, "Sales")
	require.NoError(t, err)
	require.NoError(t, s.accessSvc.SetRules(as(uOwner), uOwner, "folder", []string{folder.ID}, access.Targets{RoleIDs: []string{"role-reader"}},
		[]permissions.Action{permissions.DocsRead}, access.StateDeny))
	reader := access.WithMemo(identity.WithActor(context.Background(), identity.Actor{ID: uReader}), s.accessSvc.NewMemo())
	docList := tool(t, docs.MCPTools(s.docsSvc), "doc_list")
	reaches := func() bool {
		t.Helper()
		can, err := s.accessSvc.Can(reader, uReader, f.doc, permissions.DocsRead)
		require.NoError(t, err)
		_, getErr := s.docsSvc.Get(reader, f.doc)
		listed, _ := callPage(t, s.accessSvc, docList, uReader, `{}`)
		folders, err := s.docsSvc.ListFolders(reader, pGeneral)
		require.NoError(t, err)
		shown := false
		for _, fl := range folders {
			shown = shown || fl.ID == folder.ID
		}
		assert.Equal(t, can, getErr == nil, "a read answers as the check")
		assert.Equal(t, can, assert.ObjectsAreEqual([]string{f.doc}, listed), "doc_list answers as the check")
		assert.False(t, shown, "a folder holding nothing the reader can read stays hidden")
		return can
	}

	require.True(t, reaches(), "in Main, the project's answer")
	_, err = s.docsSvc.MoveToFolder(as(uOwner), f.doc, folder.ID)
	require.NoError(t, err)
	assert.False(t, reaches(), "moved into Sales, the folder's Deny")
	_, err = s.docsSvc.MoveToFolder(as(uOwner), f.doc, testutil.GeneralMainFolderID)
	require.NoError(t, err)
	assert.True(t, reaches(), "moved back to Main")

	_, err = s.docsSvc.MoveToFolder(as(uOwner), f.doc, folder.ID)
	require.NoError(t, err)
	require.NoError(t, s.docsSvc.DeleteFolder(as(uOwner), folder.ID))
	moved, err := f.store.Docs.GetByID(context.Background(), f.doc)
	require.NoError(t, err)
	assert.Equal(t, testutil.GeneralMainFolderID, moved.FolderID)
	assert.True(t, reaches(), "the deleted folder's rules went with it")
	rules, err := f.store.Access.ListByResource(context.Background(), "folder", folder.ID)
	require.NoError(t, err)
	assert.Empty(t, rules)
	_, err = s.accessSvc.ListRules(as(uOwner), uOwner, "folder", folder.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// TestFolderRules_GoWithTheirProject: deleting a project deletes the rules on its folders.
func TestFolderRules_GoWithTheirProject(t *testing.T) {
	f := newPermFixture(t)
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, f.store.Projects.Create(ctx, &workspace.Project{ID: "p-gone", Name: "Gone", Prefix: "GON", WorkspaceID: wsDefault, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Docs.CreateFolder(ctx, &docs.Folder{ID: "f-gone", ProjectID: "p-gone", Name: "Sales", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Access.Set(ctx, "folder", "f-gone", uReader, nil, grant("docs:read")))
	require.NoError(t, f.store.Projects.Delete(ctx, "p-gone"))
	rules, err := f.store.Access.ListByResource(ctx, "folder", "f-gone")
	require.NoError(t, err)
	assert.Empty(t, rules)
}

// TestFolderRules_FramesReachWhomTheyName: a folder rule's frame reaches whom it names and who manages the folder.
func TestFolderRules_FramesReachWhomTheyName(t *testing.T) {
	f := newPermFixture(t)
	folder, err := f.svc.docsSvc.CreateFolder(as(uOwner), pGeneral, "Sales")
	require.NoError(t, err)
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	person := access.GrantEvent{ResourceType: "folder", ResourceID: folder.ID, Target: access.TargetPerson, UserID: uPlain, ActorID: uOwner}
	assert.True(t, a.allows(as(uPlain), access.TopicGrantChanged, person), "the person themselves")
	assert.True(t, a.allows(as(uOwner), access.TopicGrantChanged, person), "who manages the folder's access")
	assert.False(t, a.allows(as(uReader), access.TopicGrantChanged, person))
	role := access.GrantEvent{ResourceType: "folder", ResourceID: folder.ID, Target: access.TargetRole, RoleID: "role-reader", ActorID: uOwner}
	assert.True(t, a.allows(as(uReader), access.TopicGrantChanged, role), "a holder of the role")
	assert.False(t, a.allows(as(uWriter), access.TopicGrantChanged, role))
}
