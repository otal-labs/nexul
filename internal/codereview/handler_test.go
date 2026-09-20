package codereview

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeReview(t *testing.T, rec *httptest.ResponseRecorder) *CodeReview {
	t.Helper()
	var rv CodeReview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rv))
	return &rv
}

func TestReviewsHandler_ListByTicket(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo, newFakeBus())
	h := NewHandler(s).Routes()
	created, err := s.Create(context.Background(), "acme/app", 42)
	require.NoError(t, err)
	repo.link("t-1", "acme/app", 42)

	t.Run("lists reviews for a ticket", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews?ticket_id=t-1", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*CodeReview
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
		assert.Equal(t, created.ID, list[0].ID)
	})
	t.Run("blank ticket_id is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestReviewsHandler_ListByPR(t *testing.T) {
	s := newTestService(newFakeRepo(), newFakeBus())
	h := NewHandler(s).Routes()
	_, err := s.Create(context.Background(), "acme/app", 42)
	require.NoError(t, err)
	_, err = s.Create(context.Background(), "acme/app", 43)
	require.NoError(t, err)

	t.Run("lists reviews for a pr", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews?repo=acme/app&number=42", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var list []*CodeReview
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list, 1)
	})
	t.Run("missing number is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews?repo=acme/app", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("non-numeric number is 400", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews?repo=acme/app&number=abc", "")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestReviewsHandler_Get(t *testing.T) {
	s := newTestService(newFakeRepo(), newFakeBus())
	h := NewHandler(s).Routes()
	created, err := s.Create(context.Background(), "acme/app", 42)
	require.NoError(t, err)

	t.Run("gets a review by id", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews/"+created.ID, "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, created.ID, decodeReview(t, rec).ID)
	})
	t.Run("missing review is 404", func(t *testing.T) {
		rec := serve(t, h, http.MethodGet, "/api/reviews/nope", "")
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestReviewsHandler_PATCHRoute_Removed(t *testing.T) {
	s := newTestService(newFakeRepo(), newFakeBus())
	h := NewHandler(s).Routes()
	created, err := s.Create(context.Background(), "acme/app", 42)
	require.NoError(t, err)

	// Provider events are the only writer; the manual PATCH route is gone.
	rec := serve(t, h, http.MethodPatch, "/api/reviews/"+created.ID, `{"status":"approved"}`)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
