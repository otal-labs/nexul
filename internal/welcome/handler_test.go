package welcome

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
)

func TestHandler_ActsOnTheCallersOwnSteps(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"] = []string{"github", "computer"}
	h := NewHandler(newTestService(repo, w)).Routes()
	call := func(method, path, actor string) (int, Progress) {
		req := httptest.NewRequest(method, path, nil)
		if actor != "" {
			req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{ID: actor}))
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var p Progress
		if rec.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		}
		return rec.Code, p
	}

	code, p := call(http.MethodPost, "/api/me/welcome/github/skip", "u-sam")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, map[string]State{"github": Skipped, "computer": Left}, states(p))
	code, p = call(http.MethodDelete, "/api/me/welcome/github/skip", "u-sam")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, map[string]State{"github": Left, "computer": Left}, states(p))
	code, p = call(http.MethodPost, "/api/me/welcome/first-run", "u-sam")
	require.Equal(t, http.StatusOK, code)
	assert.True(t, p.FirstRunDone)
	code, p = call(http.MethodGet, "/api/me/welcome", "u-sam")
	require.Equal(t, http.StatusOK, code)
	assert.True(t, p.Dot)

	code, _ = call(http.MethodPost, "/api/me/welcome/phone/skip", "u-sam")
	assert.Equal(t, http.StatusNotFound, code, "a step none of their roles gives")
	code, _ = call(http.MethodGet, "/api/me/welcome", "")
	assert.Equal(t, http.StatusUnauthorized, code)
}
