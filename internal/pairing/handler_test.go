package pairing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestHandler_PairAndList(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "one-time",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.Equal(t, "Home", created.Name)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var listed struct {
		Computers []Computer `json:"computers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Computers, 1)
	assert.Equal(t, created.ID, listed.Computers[0].ID)
}

func TestHandler_Pair_MissingBody(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	rec := doRequest(h.Routes(), http.MethodPost, "/api/pairing/computers", "u1", pairRequest{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Repair(t *testing.T) {
	t.Parallel()
	h, _, exch := newTestHandler()
	routes := h.Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok1",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	exch.version = "0.0.35"
	rec = doRequest(routes, http.MethodPost, "/api/pairing/computers/"+created.ID+"/repair", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok2",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var repaired Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &repaired))
	assert.Equal(t, created.ID, repaired.ID)
	assert.Equal(t, "0.0.35", repaired.HarnessVersion)
}

func TestHandler_DeleteComputer(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
	})
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = doRequest(routes, http.MethodDelete, "/api/pairing/computers/"+created.ID, "u1", nil)
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

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
	})
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
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

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
	})
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = doRequest(routes, http.MethodGet, "/api/pairing/projects/proj-1", "u1", nil)
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

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u2", pairRequest{
		Name: "Their box", ServerURL: "https://home.example.com", Token: "tok",
	})
	var theirs Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &theirs))

	rec = doRequest(routes, http.MethodPut, "/api/pairing/projects/proj-1", "u1", projectLinkRequest{
		ComputerID: theirs.ID, HarnessProjectID: "p",
	})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_NoIdentity_Unauthorized(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	rec := doRequest(h.Routes(), http.MethodGet, "/api/pairing/computers", "", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_Resolve_Success(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
	})
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
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
		doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
			Name: "Home", ServerURL: "https://home.example.com", Token: "tok1",
		})
		doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
			Name: "VPS", ServerURL: "https://vps.example.com", Token: "tok2",
		})

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
		rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
			Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
		})
		var created Computer
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{DefaultComputerID: created.ID})
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

		rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
			Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
		})
		var created Computer
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
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

	rec := doRequest(routes, http.MethodPost, "/api/pairing/computers", "u1", pairRequest{
		Name: "Home", ServerURL: "https://home.example.com", Token: "tok",
	})
	var created Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
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

func TestHandler_Resolve_NoIdentity_Unauthorized(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	rec := doRequest(h.Routes(), http.MethodGet, "/api/pairing/resolve", "", nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
