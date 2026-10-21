package pairing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

func newTestHandler() (*Handler, *fakeRepo, *fakeExchanger) {
	repo := newFakeRepo()
	exch := &fakeExchanger{result: harness.PairResult{BearerToken: "bearer", ExpiresIn: time.Hour}, version: "0.0.34"}
	svc := newTestService(repo, exch)
	return NewHandler(svc), repo, exch
}

func doRequest(h http.Handler, method, path, userID string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if userID != "" {
		req = req.WithContext(identity.WithActor(context.Background(), identity.Actor{ID: userID}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandler_List(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(h.Routes(), http.MethodGet, "/api/pairing/computers", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed struct {
		Computers []Computer `json:"computers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Computers, 1)
	assert.Equal(t, created.ID, listed.Computers[0].ID)
}

func TestHandler_RetiredPairingRoutes_AnswerGone(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	created := pairedComputer(t, h, "u1", "Home")
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/api/pairing/computers"},
		{http.MethodPost, "/api/pairing/computers/tunnel"},
		{http.MethodGet, "/api/pairing/computers/" + created.ID + "/tunnel/status"},
		{http.MethodGet, "/api/pairing/computers/" + created.ID + "/tunnel/token"},
		{http.MethodPost, "/api/pairing/computers/" + created.ID + "/repair"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			t.Parallel()
			rec := doRequest(h.Routes(), route.method, route.path, "u1", map[string]string{"name": "Home", "token": "tok"})
			require.Equal(t, http.StatusGone, rec.Code)
			assert.JSONEq(t, `{"message":"Pairing through a tunnel was retired; add the computer with the Nexul app","code":"GONE"}`, rec.Body.String())
		})
	}
}

func TestHandler_DeleteComputer(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(routes, http.MethodDelete, "/api/pairing/computers/"+created.ID, "u1", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers", "u1", nil)
	var listed struct {
		Computers []Computer `json:"computers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Empty(t, listed.Computers)
}

func TestHandler_Defaults_GetAndSet(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
		DefaultComputerID: created.ID, FallbackProjectID: "proj-1", Provider: "claude", Model: "sonnet",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/defaults", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var d Defaults
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
	assert.Equal(t, created.ID, d.DefaultComputerID)
	assert.Equal(t, "claude", d.Provider)
}

func TestHandler_ProjectLink_GetSetClear(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(routes, http.MethodGet, "/api/pairing/projects/proj-1", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var empty ProjectLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &empty))
	assert.Empty(t, empty.ComputerID, "unlinked project is a zero value, not a 404")

	rec = doRequest(routes, http.MethodPut, "/api/pairing/projects/proj-1", "u1", projectLinkRequest{
		ComputerID: created.ID, HarnessProjectID: "t3-proj-1", Provider: "claude", Model: "sonnet",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var link ProjectLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &link))
	assert.Equal(t, created.ID, link.ComputerID)
	assert.Equal(t, "t3-proj-1", link.HarnessProjectID)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/projects/proj-1", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &link))
	assert.Equal(t, created.ID, link.ComputerID)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/projects", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"links":[{"project_id":"proj-1","computer_id":"`+created.ID+`","harness_project_id":"t3-proj-1","provider":"claude","model":"sonnet","updated_at":"`+link.UpdatedAt.Format(time.RFC3339Nano)+`"}]}`, rec.Body.String())

	rec = doRequest(routes, http.MethodDelete, "/api/pairing/projects/proj-1", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/projects/proj-1", "u1", nil)
	var afterClear ProjectLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &afterClear))
	assert.Empty(t, afterClear.ComputerID)
}

func TestHandler_ProjectLink_ForeignComputerRejected(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	theirs := pairedComputer(t, h, "u2", "Their box")

	rec := doRequest(routes, http.MethodPut, "/api/pairing/projects/proj-1", "u1", projectLinkRequest{
		ComputerID: theirs.ID, HarnessProjectID: "p",
	})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Resolve_Success(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")
	rec := doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
		DefaultComputerID: created.ID, FallbackProjectID: "harness-proj", Provider: "claude", Model: "sonnet",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/resolve", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, created.ID, body["computer_id"])
	assert.Equal(t, "harness-proj", body["harness_project_id"])
	assert.Equal(t, "claude", body["provider"])
	assert.Equal(t, "sonnet", body["model"])
	assert.NotContains(t, rec.Body.String(), "bearer", "the resolved bearer token must never reach a response")
}

func TestHandler_Resolve_NotConfiguredReasons(t *testing.T) {
	t.Parallel()

	t.Run("unpaired", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		rec := doRequest(h.Routes(), http.MethodGet, "/api/pairing/resolve", "u1", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, false, body["ok"])
		assert.Equal(t, string(ReasonUnpaired), body["reason"])
	})

	t.Run("no_default_computer", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		routes := h.Routes()
		pairedComputer(t, h, "u1", "Home")
		pairedComputer(t, h, "u1", "VPS")

		rec := doRequest(routes, http.MethodGet, "/api/pairing/resolve", "u1", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, false, body["ok"])
		assert.Equal(t, string(ReasonNoDefaultComputer), body["reason"])
	})

	t.Run("no_default", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		routes := h.Routes()
		created := pairedComputer(t, h, "u1", "Home")
		rec := doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{DefaultComputerID: created.ID})
		require.Equal(t, http.StatusOK, rec.Code)

		rec = doRequest(routes, http.MethodGet, "/api/pairing/resolve", "u1", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, false, body["ok"])
		assert.Equal(t, string(ReasonNoDefault), body["reason"])
	})

	t.Run("expired_token", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}, version: "0.0.34"}
		svc := newTestService(repo, exch)
		routes := NewHandler(svc).Routes()

		created := pairedComputer(t, NewHandler(svc), "u1", "Home")
		rec := doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
			DefaultComputerID: created.ID, FallbackProjectID: "p",
		})
		require.Equal(t, http.StatusOK, rec.Code)

		later := NewService(Config{
			Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey,
			Now: func() time.Time { return time.Date(2026, 8, 26, 14, 0, 0, 0, time.UTC) },
		})
		rec = doRequest(NewHandler(later).Routes(), http.MethodGet, "/api/pairing/resolve", "u1", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, false, body["ok"])
		assert.Equal(t, string(ReasonExpiredToken), body["reason"])
	})
}

func TestHandler_Resolve_OnlyLeaksCallersOwnPairing(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")
	rec := doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
		DefaultComputerID: created.ID, FallbackProjectID: "p",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/resolve", "u2", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, false, body["ok"], "u2 has no pairing of their own, regardless of u1's")
	assert.Equal(t, string(ReasonUnpaired), body["reason"])
}

func TestHandler_NoIdentity_Unauthorized(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"/api/pairing/computers", "/api/pairing/resolve"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			h, _, _ := newTestHandler()
			rec := doRequest(h.Routes(), http.MethodGet, path, "", nil)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

// pairedComputer pairs one of userID's computers named name through the handler's service.
func pairedComputer(t *testing.T, h *Handler, userID, name string) Computer {
	t.Helper()
	c, err := pairAt(h.svc, t.Context(), userID, harness.KindT3Code, name, "https://"+strings.ToLower(name)+".example.com", "tok")
	require.NoError(t, err)
	return *c
}
