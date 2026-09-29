package repository

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

func serve(t *testing.T, s *fakeScanner, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	NewHandler(s, s).Routes().ServeHTTP(rec, r)
	return rec
}

func TestHandler_Scan(t *testing.T) {
	t.Run("scans and returns the result", func(t *testing.T) {
		s := &fakeScanner{
			resolvedRef: "main",
			tree:        []TreeEntry{{Path: "Dockerfile", Type: "blob"}},
			files:       map[string][]byte{"Dockerfile": []byte(rootDockerfile)},
		}
		rec := serve(t, s, http.MethodPost, "/api/repositories/scan", `{"owner":"acme","name":"app"}`)
		require.Equal(t, http.StatusOK, rec.Code)
		var result ScanResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		assert.Equal(t, "main", result.DefaultBranch)
		require.Len(t, result.Candidates, 1)
	})
	t.Run("invalid json is a 400", func(t *testing.T) {
		rec := serve(t, &fakeScanner{}, http.MethodPost, "/api/repositories/scan", `not json`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("missing owner is a 400", func(t *testing.T) {
		rec := serve(t, &fakeScanner{}, http.MethodPost, "/api/repositories/scan", `{"name":"app"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("not found scanner error is a 404", func(t *testing.T) {
		s := &fakeScanner{treeErr: apperrors.ErrNotFound}
		rec := serve(t, s, http.MethodPost, "/api/repositories/scan", `{"owner":"acme","name":"app"}`)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("lists installation repos", func(t *testing.T) {
		s := &fakeScanner{repos: []Repo{{Owner: "acme", Name: "app", FullName: "acme/app"}}}
		rec := serve(t, s, http.MethodGet, "/api/repositories", "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Repositories []Repo `json:"repositories"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Repositories, 1)
		assert.Equal(t, "acme/app", body.Repositories[0].FullName)
	})
	t.Run("scanner error is a 500", func(t *testing.T) {
		s := &fakeScanner{listErr: assertError{}}
		rec := serve(t, s, http.MethodGet, "/api/repositories", "")
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHandler_Installations(t *testing.T) {
	t.Run("lists installations with the selected count only where GitHub gives one", func(t *testing.T) {
		three := 3
		s := &fakeScanner{installs: []Installation{
			{ID: 1, AccountLogin: "octo-org", AccountType: "organization", RepositorySelection: "selected", RepositoryCount: &three},
			{ID: 2, AccountLogin: "octocat", AccountType: "user", RepositorySelection: "all"},
		}}
		rec := serve(t, s, http.MethodGet, "/api/repositories/installations", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"installations":[
			{"id":1,"account_login":"octo-org","account_type":"organization","account_avatar_url":"","repository_selection":"selected","repository_count":3,"html_url":""},
			{"id":2,"account_login":"octocat","account_type":"user","account_avatar_url":"","repository_selection":"all","html_url":""}]}`,
			rec.Body.String())
	})
	t.Run("no installations is an empty list, not null", func(t *testing.T) {
		rec := serve(t, &fakeScanner{}, http.MethodGet, "/api/repositories/installations", "")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"installations":[]}`, rec.Body.String())
	})
}

type assertError struct{}

func (assertError) Error() string { return "boom" }
