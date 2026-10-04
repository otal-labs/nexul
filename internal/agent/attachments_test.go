package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
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

func anyImage(string) bool { return true }

func TestBodyImages(t *testing.T) {
	reader := newFakeAttachmentReader()
	reader.items["img-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte("bytes")}
	reader.items["pdf-1"] = StoredAttachment{Name: "spec.pdf", MIME: "application/pdf", Bytes: []byte("bytes")}
	reader.items["huge-1"] = StoredAttachment{Name: "huge.png", MIME: "image/png", Bytes: make([]byte, MaxAttachmentBytes+1)}
	reader.err["missing-1"] = errors.New("attachment not found")
	shot := &harness.Attachment{Name: "shot.png", MIME: "image/png", Bytes: []byte("bytes")}

	tests := []struct {
		name     string
		markdown string
		want     []bodyImage
	}{
		{"an image is attached", "look ![shot](/api/attachments/img-1)", []bodyImage{{"shot.png", "/api/attachments/img-1", shot}}},
		{"an image over the size cap is left to its link", "![big](/api/attachments/huge-1)",
			[]bodyImage{{"huge.png", "/api/attachments/huge-1", nil}}},
		{"a file that is not an image is left out", "see [spec](/api/attachments/pdf-1)", nil},
		{"an attachment that cannot be read is left out", "![x](/api/attachments/missing-1)", nil},
		{"no reference", "plain text, nothing to fetch", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, bodyImages(t.Context(), tt.markdown, reader, anyImage))
		})
	}
}

func TestBodyImages_ImageNamedTwice_IsReadAndAttachedOnce(t *testing.T) {
	reader := newFakeAttachmentReader()
	reader.items["img-1"] = StoredAttachment{Name: "shot.png", MIME: "image/png", Bytes: []byte("x")}

	got := bodyImages(t.Context(), "![a](/api/attachments/img-1) and ![b](/api/attachments/img-1)", reader, anyImage)

	require.Len(t, got, 1)
	assert.Equal(t, 1, reader.calls["img-1"])
}

func TestBodyImages_OverTheTurnsTotal_LaterImagesAreLeftToTheirLinks(t *testing.T) {
	reader := newFakeAttachmentReader()
	full := make([]byte, MaxAttachmentBytes)
	for _, id := range []string{"img-1", "img-2", "img-3"} {
		reader.items[id] = StoredAttachment{Name: id + ".png", MIME: "image/png", Bytes: full}
	}

	got := bodyImages(t.Context(), "![a](/api/attachments/img-1) ![b](/api/attachments/img-2) ![c](/api/attachments/img-3)", reader, anyImage)

	require.Len(t, got, 3)
	assert.NotNil(t, got[0].attachment)
	assert.NotNil(t, got[1].attachment)
	assert.Nil(t, got[2].attachment, "the third image would pass the turn's total")
}
