package gitprovider

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newGitHandler(p GitProvider) http.Handler {
	return NewHandler(p).Routes()
}

func prsHandler(t *testing.T) http.Handler {
	return newGitHandler(&fakeProvider{prs: []*PR{
		{Number: 1, Title: "Fix login", State: PRStateOpen, Author: "alice", LinkedTicketIDs: []string{"42"}},
		{Number: 2, Title: "Wire FTS", State: PRStateOpen, Author: "bob", LinkedTicketIDs: []string{"42", "43"}},
	}})
}

func TestGitHandler_ListPRs(t *testing.T) {
	t.Run("lists pull requests with defaults", func(t *testing.T) {
		rec := serve(t, prsHandler(t), http.MethodGet, "/api/repos/acme/app/prs")
		require.Equal(t, http.StatusOK, rec.Code)
		var prs []*PR
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prs))
		require.Len(t, prs, 2)
		assert.Equal(t, 2, prs[1].Number)
	})
	t.Run("provider error is a 500", func(t *testing.T) {
		h := newGitHandler(&fakeProvider{err: errors.New("github down")})
		rec := serve(t, h, http.MethodGet, "/api/repos/acme/app/prs")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestGitHandler_GetPR(t *testing.T) {
	t.Run("gets a single pull request", func(t *testing.T) {
		h := newGitHandler(&fakeProvider{pr: &PR{Number: 1, Title: "Fix login"}})
		rec := serve(t, h, http.MethodGet, "/api/repos/acme/app/prs/1")
		require.Equal(t, http.StatusOK, rec.Code)
		var pr PR
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &pr))
		assert.Equal(t, 1, pr.Number)
	})
	t.Run("non-numeric number is 400", func(t *testing.T) {
		rec := serve(t, prsHandler(t), http.MethodGet, "/api/repos/acme/app/prs/abc")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestGitHandler_GetRepo(t *testing.T) {
	t.Run("gets a repo", func(t *testing.T) {
		h := newGitHandler(&fakeProvider{repo: &Repo{Owner: "acme", Name: "app"}})
		rec := serve(t, h, http.MethodGet, "/api/repos/acme/app")
		require.Equal(t, http.StatusOK, rec.Code)
		var repo Repo
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &repo))
		assert.Equal(t, "acme", repo.Owner)
	})
}

func TestGitHandler_ListPRs_LinkedTicketCount(t *testing.T) {
	rec := serve(t, prsHandler(t), http.MethodGet, "/api/repos/acme/app/prs")
	require.Equal(t, http.StatusOK, rec.Code)
	var prs []*PR
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prs))
	assert.Len(t, prs[1].LinkedTicketIDs, 2, "the review badge counts linked tickets from the pr payload")
}
