package composite

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/attachments"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR")

type attachmentRepo struct {
	items map[string]*attachments.Attachment
}

func (r *attachmentRepo) Create(_ context.Context, a *attachments.Attachment) error {
	r.items[a.ID] = a
	return nil
}

func (r *attachmentRepo) GetByID(_ context.Context, id string) (*attachments.Attachment, error) {
	a, ok := r.items[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return a, nil
}

func (r *attachmentRepo) ListByOwner(context.Context, attachments.Owner) ([]*attachments.Attachment, error) {
	return nil, nil
}

func (r *attachmentRepo) Delete(context.Context, string) error { return nil }

// ticketFiles lets the caller read every ticket's files and change only those of the tickets not in readOnly.
type ticketFiles struct {
	readOnly map[string]bool
}

func (g ticketFiles) RequireTicket(_ context.Context, ticketID string, action permissions.Action) error {
	if action == permissions.TicketsWrite && g.readOnly[ticketID] {
		return fmt.Errorf("%w: no %s on ticket %s", apperrs.ErrForbidden, action, ticketID)
	}
	return nil
}

type attachmentFixture struct {
	fixture
	repo  *attachmentRepo
	tools []mcptool.Tool
}

// newAttachmentFixture holds chat's shot.png on a ticket the caller may only read, t-2 (REF-2).
func newAttachmentFixture(t *testing.T) attachmentFixture {
	t.Helper()
	f := newFixture(t)
	repo := &attachmentRepo{items: map[string]*attachments.Attachment{
		"a-shot": {ID: "a-shot", TicketID: "t-2", Name: "shot.png", ContentType: "image/png", Size: int64(len(pngBytes)), Data: pngBytes},
	}}
	svc := attachments.NewService(repo, nil, nil)
	svc.SetTicketAccess(ticketFiles{readOnly: map[string]bool{"t-2": true}})
	return attachmentFixture{fixture: f, repo: repo, tools: AttachmentTools(svc, f.tickets)}
}

func b64(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func TestAttachmentCreate_Errors(t *testing.T) {
	tests := []struct {
		name string
		ctx  func(context.Context) context.Context
		args string
		want error
	}{
		{"neither content nor a copy", asUser, `{"ticket_id":"REF-1","name":"a.png"}`, apperrs.ErrInvalid},
		{"both content and a copy", asUser, `{"ticket_id":"REF-1","name":"a.png","content":"` + b64(pngBytes) + `","from_attachment_id":"a-shot"}`, apperrs.ErrInvalid},
		{"content without a name", asUser, `{"ticket_id":"REF-1","content":"` + b64(pngBytes) + `"}`, apperrs.ErrInvalid},
		{"content that is not base64", asUser, `{"ticket_id":"REF-1","name":"a.png","content":"not base64!"}`, apperrs.ErrInvalid},
		{"no owner", asUser, `{"name":"a.png","content":"` + b64(pngBytes) + `"}`, apperrs.ErrInvalid},
		{"an unknown ticket key", asUser, `{"ticket_id":"REF-99","name":"a.png","content":"` + b64(pngBytes) + `"}`, apperrs.ErrNotFound},
		{"a missing file to copy", asUser, `{"ticket_id":"REF-1","from_attachment_id":"nope"}`, apperrs.ErrNotFound},
		{"a ticket the caller may only read", asUser, `{"ticket_id":"REF-2","name":"a.png","content":"` + b64(pngBytes) + `"}`, apperrs.ErrForbidden},
		{"a doc without a doc access check fails closed", asUser, `{"doc_id":"d-1","name":"a.png","content":"` + b64(pngBytes) + `"}`, apperrs.ErrForbidden},
		{"nobody signed in", anonymous, `{"conversation_id":"c-1","name":"a.png","content":"` + b64(pngBytes) + `"}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAttachmentFixture(t)
			_, err := call(t, tt.ctx(t.Context()), f.tools, "attachment_create", tt.args)
			require.ErrorIs(t, err, tt.want)
			assert.Len(t, f.repo.items, 1, "nothing was stored")
		})
	}
}

func TestAttachmentCreate_UploadsOntoATicketByKeyWithMarkdownThatRendersInline(t *testing.T) {
	f := newAttachmentFixture(t)
	out, err := call(t, asUser(t.Context()), f.tools, "attachment_create", `{"ticket_id":"REF-1","name":"logo.png","content":"`+b64(pngBytes)+`"}`)
	require.NoError(t, err)
	res := out.(attachmentResult)
	stored := f.repo.items[res.ID]
	require.NotNil(t, stored)
	assert.Equal(t, "t-1", stored.TicketID)
	assert.Equal(t, "u-1", stored.UploadedBy)
	assert.Equal(t, pngBytes, stored.Data)
	assert.Equal(t, "image/png", res.ContentType)
	assert.Equal(t, "/api/attachments/"+res.ID, res.URL)
	assert.Equal(t, "![logo.png](/api/attachments/"+res.ID+")", res.Markdown)
}

func TestAttachmentCreate_AFileThatIsNotAnImageIsALink(t *testing.T) {
	f := newAttachmentFixture(t)
	out, err := call(t, asUser(t.Context()), f.tools, "attachment_create", `{"ticket_id":"t-1","name":"notes.txt","content":"`+b64([]byte("hello"))+`"}`)
	require.NoError(t, err)
	res := out.(attachmentResult)
	assert.Equal(t, "text/plain; charset=utf-8", res.ContentType)
	assert.Equal(t, "[notes.txt](/api/attachments/"+res.ID+")", res.Markdown)
}

func TestAttachmentCreate_CopiesAnExistingFileByItsLinkOnTheServer(t *testing.T) {
	f := newAttachmentFixture(t)
	out, err := call(t, asUser(t.Context()), f.tools, "attachment_create", `{"ticket_id":"REF-1","from_attachment_id":"https://nexul.example/api/attachments/a-shot"}`)
	require.NoError(t, err)
	res := out.(attachmentResult)
	assert.NotEqual(t, "a-shot", res.ID)
	assert.Equal(t, "shot.png", res.Name, "the copy keeps the source's name")
	assert.Equal(t, "t-1", f.repo.items[res.ID].TicketID)
	assert.Equal(t, pngBytes, f.repo.items[res.ID].Data)

	out, err = call(t, asUser(t.Context()), f.tools, "attachment_create", `{"ticket_id":"REF-1","from_attachment_id":"a-shot","name":"logo.png"}`)
	require.NoError(t, err)
	assert.Equal(t, "logo.png", out.(attachmentResult).Name)
}

func TestAttachmentGet(t *testing.T) {
	t.Run("a missing attachment says where ids come from", func(t *testing.T) {
		_, err := call(t, asUser(t.Context()), newAttachmentFixture(t).tools, "attachment_get", `{"id":"nope"}`)
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		assert.Contains(t, err.Error(), "attachment_create")
	})
	t.Run("nobody signed in", func(t *testing.T) {
		_, err := call(t, t.Context(), newAttachmentFixture(t).tools, "attachment_get", `{"id":"a-shot"}`)
		require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("returns the bytes and metadata, by id or by the link in a body", func(t *testing.T) {
		for _, ref := range []string{"a-shot", "/api/attachments/a-shot"} {
			out, err := call(t, asUser(t.Context()), newAttachmentFixture(t).tools, "attachment_get", `{"id":"`+ref+`"}`)
			require.NoError(t, err)
			file := out.(mcptool.File)
			assert.Equal(t, "image/png", file.MIMEType)
			assert.Equal(t, pngBytes, file.Data)
			assert.Equal(t, "attachments://a-shot", file.URI)
			assert.Equal(t, "![shot.png](/api/attachments/a-shot)", file.Meta.(attachmentResult).Markdown)
		}
	})
}
