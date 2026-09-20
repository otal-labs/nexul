package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAttachmentReader is the AttachmentReader seam's fake, keyed by attachment id; calls tracks fetch counts.
type fakeAttachmentReader struct {
	items map[string]StoredAttachment
	err   map[string]error
	calls map[string]int
}

func newFakeAttachmentReader() *fakeAttachmentReader {
	return &fakeAttachmentReader{items: map[string]StoredAttachment{}, err: map[string]error{}, calls: map[string]int{}}
}

func (f *fakeAttachmentReader) Get(_ context.Context, id string) (StoredAttachment, error) {
	f.calls[id]++
	if err, ok := f.err[id]; ok {
		return StoredAttachment{}, err
	}
	return f.items[id], nil
}

func TestExtractAttachments(t *testing.T) {
	reader := newFakeAttachmentReader()
	reader.items["img-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte("bytes")}
	reader.items["pdf-1"] = StoredAttachment{Name: "spec.pdf", MIME: "application/pdf", Bytes: []byte("bytes")}
	reader.items["huge-1"] = StoredAttachment{Name: "huge.png", MIME: "image/png", Bytes: make([]byte, MaxAttachmentBytes+1)}
	reader.err["missing-1"] = errors.New("attachment not found")

	tests := []struct {
		name     string
		markdown string
		wantBody string
		wantAtts int
	}{
		{"one image", "look ![shot](/api/attachments/img-1)", "look [image: shot.png, attached to this turn]", 1},
		{"pdf attachment", "see [spec](/api/attachments/pdf-1)", "see [attachment omitted: spec.pdf]", 0},
		{"oversized image", "![big](/api/attachments/huge-1)", "[attachment omitted: huge.png]", 0},
		{"fetch error", "![x](/api/attachments/missing-1)", "[attachment omitted: x]", 0},
		{"no reference", "plain text, nothing to fetch", "plain text, nothing to fetch", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, atts := ExtractAttachments(t.Context(), tt.markdown, reader, NewAttachmentBudget())
			assert.Equal(t, tt.wantBody, got)
			assert.Len(t, atts, tt.wantAtts)
		})
	}
}

func TestExtractAttachments_DuplicateID_FetchesOnce(t *testing.T) {
	reader := newFakeAttachmentReader()
	reader.items["img-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte("x")}

	got, atts := ExtractAttachments(t.Context(), "![a](/api/attachments/img-1) and ![b](/api/attachments/img-1)", reader, NewAttachmentBudget())

	assert.Equal(t, "[image: shot.png, attached to this turn] and [image: shot.png, attached to this turn]", got)
	require.Len(t, atts, 1)
	assert.Equal(t, 1, reader.calls["img-1"])
}

func TestExtractAttachments_PerTurnBudget_SecondImageOmitted(t *testing.T) {
	reader := newFakeAttachmentReader()
	reader.items["img-1"] = StoredAttachment{Name: "one.png", MIME: "image/png", Bytes: make([]byte, 5)}
	reader.items["img-2"] = StoredAttachment{Name: "two.png", MIME: "image/png", Bytes: make([]byte, 5)}
	budget := &AttachmentBudget{Remaining: 6}

	got, atts := ExtractAttachments(t.Context(), "![a](/api/attachments/img-1) ![b](/api/attachments/img-2)", reader, budget)

	assert.Equal(t, "[image: one.png, attached to this turn] [attachment omitted: two.png]", got)
	require.Len(t, atts, 1)
	assert.Equal(t, "one.png", atts[0].Name)
	assert.Equal(t, 1, budget.Remaining)
}

func TestExtractAttachments_SharedBudgetAcrossCalls(t *testing.T) {
	reader := newFakeAttachmentReader()
	reader.items["img-1"] = StoredAttachment{Name: "one.png", MIME: "image/png", Bytes: make([]byte, 5)}
	reader.items["img-2"] = StoredAttachment{Name: "two.png", MIME: "image/png", Bytes: make([]byte, 5)}
	budget := &AttachmentBudget{Remaining: 6}

	first, atts1 := ExtractAttachments(t.Context(), "![a](/api/attachments/img-1)", reader, budget)
	second, atts2 := ExtractAttachments(t.Context(), "![b](/api/attachments/img-2)", reader, budget)

	assert.Equal(t, "[image: one.png, attached to this turn]", first)
	assert.Equal(t, "[attachment omitted: two.png]", second)
	assert.Len(t, atts1, 1)
	assert.Empty(t, atts2)
}

func TestExtractAttachments_NilReader_LeavesMarkdownUnchanged(t *testing.T) {
	got, atts := ExtractAttachments(t.Context(), "![x](/api/attachments/img-1)", nil, nil)

	assert.Equal(t, "![x](/api/attachments/img-1)", got)
	assert.Nil(t, atts)
}
