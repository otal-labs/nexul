package attachments

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var fixedNow = time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

// pngBytes is the PNG magic header, enough for http.DetectContentType to say image/png.
var pngBytes = []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16))

type fakeRepo struct {
	items     map[string]*Attachment
	order     []string
	createErr error
	getErr    error
	listErr   error
	deleteErr error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{items: map[string]*Attachment{}} }

func (f *fakeRepo) Create(_ context.Context, a *Attachment) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.items[a.ID] = a
	f.order = append(f.order, a.ID)
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (*Attachment, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	a, ok := f.items[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return a, nil
}

func (f *fakeRepo) ListByOwner(_ context.Context, owner Owner) ([]*Attachment, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*Attachment
	for _, id := range f.order {
		if a := f.items[id]; a.Owner() == owner {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeRepo) Delete(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.items[id]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

type fakeAccess struct {
	can    bool
	canErr error
	calls  []permissions.Action
}

func (f *fakeAccess) Can(_ context.Context, _, _ string, action permissions.Action) (bool, error) {
	f.calls = append(f.calls, action)
	return f.can, f.canErr
}

type fakeMemoryAccess struct {
	can    bool
	canErr error
	calls  []permissions.Action
}

func (f *fakeMemoryAccess) Can(_ context.Context, _, _ string, action permissions.Action) (bool, error) {
	f.calls = append(f.calls, action)
	return f.can, f.canErr
}

func newTestService(repo *fakeRepo, access AccessChecker) *Service {
	return newTestServiceWithMemoryAccess(repo, access, &fakeMemoryAccess{can: true})
}

func newTestServiceWithMemoryAccess(repo *fakeRepo, access AccessChecker, memoryAccess MemoryAccessChecker) *Service {
	s := NewService(repo, access, memoryAccess)
	s.now = func() time.Time { return fixedNow }
	return s
}

func testCtx() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "user-1"})
}

func TestUpload_Validation(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		owner   Owner
		data    []byte
		wantErr error
	}{
		{"no actor", context.Background(), Owner{TicketID: "t-1"}, pngBytes, apperrs.ErrUnauthorized},
		{"no owner", testCtx(), Owner{}, pngBytes, apperrs.ErrInvalid},
		{"both owners", testCtx(), Owner{DocID: "d-1", TicketID: "t-1"}, pngBytes, apperrs.ErrInvalid},
		{"doc and conversation", testCtx(), Owner{DocID: "d-1", ConversationID: "c-1"}, pngBytes, apperrs.ErrInvalid},
		{"ticket and conversation", testCtx(), Owner{TicketID: "t-1", ConversationID: "c-1"}, pngBytes, apperrs.ErrInvalid},
		{"memory and doc", testCtx(), Owner{MemoryID: "m-1", DocID: "d-1"}, pngBytes, apperrs.ErrInvalid},
		{"all four owners", testCtx(), Owner{DocID: "d-1", TicketID: "t-1", ConversationID: "c-1", MemoryID: "m-1"}, pngBytes, apperrs.ErrInvalid},
		{"empty file", testCtx(), Owner{TicketID: "t-1"}, nil, apperrs.ErrInvalid},
		{"oversized file", testCtx(), Owner{TicketID: "t-1"}, make([]byte, MaxSize+1), apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(newFakeRepo(), &fakeAccess{can: true})
			_, err := s.Upload(tt.ctx, tt.owner, "shot.png", tt.data)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUpload_TicketAttachment_SniffsTypeAndCleansName(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, &fakeAccess{can: false})

	a, err := s.Upload(testCtx(), Owner{TicketID: "t-1"}, `C:\Users\onik97\..\shot.png`, pngBytes)
	require.NoError(t, err)
	assert.Equal(t, "image/png", a.ContentType)
	assert.Equal(t, "shot.png", a.Name)
	assert.Equal(t, int64(len(pngBytes)), a.Size)
	assert.Equal(t, "user-1", a.UploadedBy)
	assert.Equal(t, fixedNow, a.CreatedAt)
	assert.Equal(t, "t-1", a.TicketID)
	assert.True(t, a.Inline())
	assert.Len(t, repo.items, 1)
}

func TestUpload_EmptyNameFallsBack(t *testing.T) {
	s := newTestService(newFakeRepo(), &fakeAccess{can: true})
	a, err := s.Upload(testCtx(), Owner{TicketID: "t-1"}, "   ", []byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, "file", a.Name)
	assert.Equal(t, "text/plain; charset=utf-8", a.ContentType)
	assert.False(t, a.Inline())
}

func TestUpload_DocAttachment_RequiresEditBit(t *testing.T) {
	t.Run("denied", func(t *testing.T) {
		access := &fakeAccess{can: false}
		s := newTestService(newFakeRepo(), access)
		_, err := s.Upload(testCtx(), Owner{DocID: "d-1"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Equal(t, []permissions.Action{permissions.DocsWrite}, access.calls)
	})
	t.Run("checker error denies", func(t *testing.T) {
		s := newTestService(newFakeRepo(), &fakeAccess{can: true, canErr: errors.New("boom")})
		_, err := s.Upload(testCtx(), Owner{DocID: "d-1"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("nil checker fails closed", func(t *testing.T) {
		s := newTestService(newFakeRepo(), nil)
		_, err := s.Upload(testCtx(), Owner{DocID: "d-1"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("allowed", func(t *testing.T) {
		s := newTestService(newFakeRepo(), &fakeAccess{can: true})
		a, err := s.Upload(testCtx(), Owner{DocID: "d-1"}, "a.png", pngBytes)
		require.NoError(t, err)
		assert.Equal(t, "d-1", a.DocID)
	})
}

func TestUpload_MemoryAttachment_RequiresWriteBit(t *testing.T) {
	t.Run("denied", func(t *testing.T) {
		memoryAccess := &fakeMemoryAccess{can: false}
		s := newTestServiceWithMemoryAccess(newFakeRepo(), &fakeAccess{can: true}, memoryAccess)
		_, err := s.Upload(testCtx(), Owner{MemoryID: "m-1"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
		assert.Equal(t, []permissions.Action{permissions.MemoriesWrite}, memoryAccess.calls)
	})
	t.Run("checker error denies", func(t *testing.T) {
		s := newTestServiceWithMemoryAccess(newFakeRepo(), &fakeAccess{can: true}, &fakeMemoryAccess{can: true, canErr: errors.New("boom")})
		_, err := s.Upload(testCtx(), Owner{MemoryID: "m-1"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("nil checker fails closed", func(t *testing.T) {
		s := newTestServiceWithMemoryAccess(newFakeRepo(), &fakeAccess{can: true}, nil)
		_, err := s.Upload(testCtx(), Owner{MemoryID: "m-1"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("allowed", func(t *testing.T) {
		s := newTestServiceWithMemoryAccess(newFakeRepo(), &fakeAccess{can: true}, &fakeMemoryAccess{can: true})
		a, err := s.Upload(testCtx(), Owner{MemoryID: "m-1"}, "a.png", pngBytes)
		require.NoError(t, err)
		assert.Equal(t, "m-1", a.MemoryID)
	})
}

func TestCopyAttachmentsWithIDs_CopiesUnderNewIDsAndOwner(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, &fakeAccess{can: true})
	a, err := s.Upload(testCtx(), Owner{MemoryID: "m-1"}, "a.png", pngBytes)
	require.NoError(t, err)

	err = s.CopyAttachmentsWithIDs(testCtx(), Owner{MemoryID: "m-1"}, Owner{MemoryID: "m-2"}, map[string]string{a.ID: "a-new"})
	require.NoError(t, err)

	copied, err := repo.GetByID(testCtx(), "a-new")
	require.NoError(t, err)
	assert.Equal(t, "m-2", copied.MemoryID)
	assert.Equal(t, a.Name, copied.Name)
	assert.Equal(t, pngBytes, copied.Data)

	// The source is untouched.
	_, err = repo.GetByID(testCtx(), a.ID)
	require.NoError(t, err)
}

func TestListOwnerAttachmentIDs_NoPermissionCheck(t *testing.T) {
	repo := newFakeRepo()
	memoryAccess := &fakeMemoryAccess{can: false}
	s := newTestServiceWithMemoryAccess(repo, &fakeAccess{can: false}, memoryAccess)
	_, err := newTestService(repo, &fakeAccess{can: true}).Upload(testCtx(), Owner{MemoryID: "m-1"}, "a.png", pngBytes)
	require.NoError(t, err)

	ids, err := s.ListOwnerAttachmentIDs(testCtx(), Owner{MemoryID: "m-1"})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Empty(t, memoryAccess.calls)
}

// TestConversationAttachment_NoAccessCheck covers upload/list/get/delete for a conversation owner: chat v1 enforces no per-conversation permission, so an authenticated actor is all requireOwner needs, and access.Can is never called.
func TestConversationAttachment_NoAccessCheck(t *testing.T) {
	repo := newFakeRepo()
	access := &fakeAccess{can: false}
	s := newTestService(repo, access)

	a, err := s.Upload(testCtx(), Owner{ConversationID: "conv-1"}, "shot.png", pngBytes)
	require.NoError(t, err)
	assert.Equal(t, "conv-1", a.ConversationID)
	assert.Empty(t, access.calls)

	as, err := s.List(testCtx(), Owner{ConversationID: "conv-1"})
	require.NoError(t, err)
	require.Len(t, as, 1)
	assert.Empty(t, access.calls)

	got, err := s.Get(testCtx(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, pngBytes, got.Data)
	assert.Empty(t, access.calls)

	require.NoError(t, s.Delete(testCtx(), a.ID))
	assert.Empty(t, access.calls)
}

func TestUpload_RepoErrors(t *testing.T) {
	t.Run("unknown owner maps FK conflict to not found", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = apperrs.ErrConflict
		s := newTestService(repo, &fakeAccess{can: true})
		_, err := s.Upload(testCtx(), Owner{TicketID: "missing"}, "a.png", pngBytes)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("other repo error passes through", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("disk full")
		s := newTestService(repo, &fakeAccess{can: true})
		_, err := s.Upload(testCtx(), Owner{TicketID: "t-1"}, "a.png", pngBytes)
		require.Error(t, err)
		assert.NotErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestGet(t *testing.T) {
	repo := newFakeRepo()
	access := &fakeAccess{can: true}
	s := newTestService(repo, access)
	docAtt, err := s.Upload(testCtx(), Owner{DocID: "d-1"}, "a.png", pngBytes)
	require.NoError(t, err)

	t.Run("empty id", func(t *testing.T) {
		_, err := s.Get(testCtx(), " ")
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("missing", func(t *testing.T) {
		_, err := s.Get(testCtx(), "nope")
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("returns bytes with read check", func(t *testing.T) {
		access.calls = nil
		got, err := s.Get(testCtx(), docAtt.ID)
		require.NoError(t, err)
		assert.Equal(t, pngBytes, got.Data)
		assert.Equal(t, []permissions.Action{permissions.DocsRead}, access.calls)
	})
	t.Run("denied doc read", func(t *testing.T) {
		denied := newTestService(repo, &fakeAccess{can: false})
		_, err := denied.Get(testCtx(), docAtt.ID)
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("repo error", func(t *testing.T) {
		broken := newFakeRepo()
		broken.getErr = errors.New("boom")
		_, err := newTestService(broken, access).Get(testCtx(), "x")
		require.Error(t, err)
	})
}

func TestList(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, &fakeAccess{can: true})
	_, err := s.Upload(testCtx(), Owner{TicketID: "t-1"}, "a.png", pngBytes)
	require.NoError(t, err)
	_, err = s.Upload(testCtx(), Owner{TicketID: "t-2"}, "b.png", pngBytes)
	require.NoError(t, err)

	t.Run("scoped to owner", func(t *testing.T) {
		as, err := s.List(testCtx(), Owner{TicketID: "t-1"})
		require.NoError(t, err)
		require.Len(t, as, 1)
		assert.Equal(t, "a.png", as[0].Name)
	})
	t.Run("invalid owner", func(t *testing.T) {
		_, err := s.List(testCtx(), Owner{})
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("denied doc", func(t *testing.T) {
		_, err := newTestService(repo, &fakeAccess{can: false}).List(testCtx(), Owner{DocID: "d-1"})
		require.ErrorIs(t, err, apperrs.ErrForbidden)
	})
	t.Run("repo error", func(t *testing.T) {
		broken := newFakeRepo()
		broken.listErr = errors.New("boom")
		_, err := newTestService(broken, &fakeAccess{can: true}).List(testCtx(), Owner{TicketID: "t-1"})
		require.Error(t, err)
	})
}

func TestDelete(t *testing.T) {
	repo := newFakeRepo()
	access := &fakeAccess{can: true}
	s := newTestService(repo, access)
	a, err := s.Upload(testCtx(), Owner{DocID: "d-1"}, "a.png", pngBytes)
	require.NoError(t, err)

	t.Run("empty id", func(t *testing.T) {
		require.ErrorIs(t, s.Delete(testCtx(), ""), apperrs.ErrInvalid)
	})
	t.Run("missing", func(t *testing.T) {
		require.ErrorIs(t, s.Delete(testCtx(), "nope"), apperrs.ErrNotFound)
	})
	t.Run("denied doc edit", func(t *testing.T) {
		require.ErrorIs(t, newTestService(repo, &fakeAccess{can: false}).Delete(testCtx(), a.ID), apperrs.ErrForbidden)
	})
	t.Run("repo delete error", func(t *testing.T) {
		repo.deleteErr = errors.New("boom")
		require.Error(t, s.Delete(testCtx(), a.ID))
		repo.deleteErr = nil
	})
	t.Run("deletes", func(t *testing.T) {
		access.calls = nil
		require.NoError(t, s.Delete(testCtx(), a.ID))
		assert.Equal(t, []permissions.Action{permissions.DocsWrite}, access.calls)
		_, err := s.Get(testCtx(), a.ID)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})
}

func TestInline(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/svg+xml", false},
		{"text/html; charset=utf-8", false},
		{"application/pdf", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, (&Attachment{ContentType: tt.ct}).Inline(), tt.ct)
	}
}

func TestCleanName(t *testing.T) {
	assert.Equal(t, "file", cleanName(""))
	assert.Equal(t, "file", cleanName("/"))
	assert.Equal(t, "x.png", cleanName("a/b/x.png"))
	long := strings.Repeat("a", 300) + ".png"
	assert.Len(t, cleanName(long), maxNameLen)
	assert.True(t, strings.HasSuffix(cleanName(long), ".png"))
}
