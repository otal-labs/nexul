package attachments

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func newHandler(access AccessChecker) http.Handler {
	return NewHandler(newTestService(newFakeRepo(), access)).Routes()
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: "user-1"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func multipartUpload(t *testing.T, fields map[string]string, filename string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	if filename != "" {
		fw, err := mw.CreateFormFile("file", filename)
		require.NoError(t, err)
		_, err = fw.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/attachments", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func decodeAttachment(t *testing.T, rec *httptest.ResponseRecorder) *Attachment {
	t.Helper()
	var a Attachment
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &a))
	return &a
}

func TestHandler_Upload(t *testing.T) {
	h := newHandler(&fakeAccess{can: true})

	t.Run("uploads to a ticket", func(t *testing.T) {
		rec := serve(h, multipartUpload(t, map[string]string{"ticket_id": "t-1"}, "shot.png", pngBytes))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		a := decodeAttachment(t, rec)
		assert.Equal(t, "shot.png", a.Name)
		assert.Equal(t, "image/png", a.ContentType)
		assert.Equal(t, "t-1", a.TicketID)
		assert.NotEmpty(t, a.ID)
	})
	t.Run("uploads to a conversation", func(t *testing.T) {
		rec := serve(h, multipartUpload(t, map[string]string{"conversation_id": "conv-1"}, "shot.png", pngBytes))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		a := decodeAttachment(t, rec)
		assert.Equal(t, "conv-1", a.ConversationID)
	})
	t.Run("name field overrides the multipart filename", func(t *testing.T) {
		rec := serve(h, multipartUpload(t, map[string]string{"doc_id": "d-1", "name": "Pasted image.png"}, "blob", pngBytes))
		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "Pasted image.png", decodeAttachment(t, rec).Name)
	})
	t.Run("missing file is 400", func(t *testing.T) {
		rec := serve(h, multipartUpload(t, map[string]string{"ticket_id": "t-1"}, "", nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing owner is 400", func(t *testing.T) {
		rec := serve(h, multipartUpload(t, nil, "a.png", pngBytes))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("not multipart is 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/attachments", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		assert.Equal(t, http.StatusBadRequest, serve(h, req).Code)
	})
	t.Run("oversized body is 400", func(t *testing.T) {
		rec := serve(h, multipartUpload(t, map[string]string{"ticket_id": "t-1"}, "big.bin", make([]byte, MaxSize+65<<10)))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "exceeds")
	})
	t.Run("denied doc is 403", func(t *testing.T) {
		rec := serve(newHandler(&fakeAccess{can: false}), multipartUpload(t, map[string]string{"doc_id": "d-1"}, "a.png", pngBytes))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("unauthenticated is 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, multipartUpload(t, map[string]string{"ticket_id": "t-1"}, "a.png", pngBytes))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_ListServeDelete(t *testing.T) {
	h := newHandler(&fakeAccess{can: true})
	img := decodeAttachment(t, serve(h, multipartUpload(t, map[string]string{"ticket_id": "t-1"}, "shot.png", pngBytes)))
	svg := decodeAttachment(t, serve(h, multipartUpload(t, map[string]string{"ticket_id": "t-1"}, "logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))))

	t.Run("lists by owner without bytes", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments?ticket_id=t-1", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		var as []Attachment
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &as))
		require.Len(t, as, 2)
		assert.Equal(t, "shot.png", as[0].Name)
		assert.NotContains(t, rec.Body.String(), `"data"`)
	})
	t.Run("lists by conversation_id", func(t *testing.T) {
		serve(h, multipartUpload(t, map[string]string{"conversation_id": "conv-1"}, "chat.png", pngBytes))
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments?conversation_id=conv-1", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		var as []Attachment
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &as))
		require.Len(t, as, 1)
		assert.Equal(t, "chat.png", as[0].Name)
	})
	t.Run("empty list is [] not null", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments?ticket_id=t-9", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "[]", rec.Body.String())
	})
	t.Run("list without owner is 400", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments", nil))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("serves an image inline with nosniff", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments/"+img.ID, nil))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
		assert.Equal(t, `inline; filename=shot.png`, rec.Header().Get("Content-Disposition"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Contains(t, rec.Header().Get("Cache-Control"), "immutable")
		body, _ := io.ReadAll(rec.Body)
		assert.Equal(t, pngBytes, body)
	})
	t.Run("serves svg as a download", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments/"+svg.ID, nil))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Content-Disposition"), "attachment;"), rec.Header().Get("Content-Disposition"))
	})
	t.Run("missing is 404", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/attachments/nope", nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("deletes", func(t *testing.T) {
		rec := serve(h, httptest.NewRequest(http.MethodDelete, "/api/attachments/"+img.ID, nil))
		assert.Equal(t, http.StatusNoContent, rec.Code)
		rec = serve(h, httptest.NewRequest(http.MethodDelete, "/api/attachments/"+img.ID, nil))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
