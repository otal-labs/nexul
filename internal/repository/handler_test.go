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

func serve(t *testing.T, s Scanner, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	NewHandler(s).Routes().ServeHTTP(rec, r)
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

type assertError struct{}

func (assertError) Error() string { return "boom" }
