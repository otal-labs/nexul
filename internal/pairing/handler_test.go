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

	rec := doRequest(routes, http.MethodGet, "/api/pairing/defaults", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var d Defaults
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
	assert.Equal(t, PlacementInOrder, d.Placement, "runs go in order until the person says otherwise")

	rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{
		Provider: "claude", Model: "sonnet", Placement: PlacementSpread, CloneAutomatically: true,
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/defaults", "u1", nil)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
	assert.Equal(t, Defaults{Provider: "claude", Model: "sonnet", Placement: PlacementSpread, CloneAutomatically: true}, d)

	rec = doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{Placement: "random"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_ProjectLink_GetSetClear(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(routes, http.MethodGet, "/api/pairing/projects/proj-1", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"project_id":"proj-1","computers":[],"checkouts":[],"updated_at":"0001-01-01T00:00:00Z"}`, rec.Body.String(),
		"an unlinked project is a zero value that follows the order, not a 404")

	rec = doRequest(routes, http.MethodPut, "/api/pairing/projects/proj-1", "u1", projectLinkRequest{
		Provider: "claude", Model: "sonnet", Placement: PlacementSpread, Computers: []string{created.ID},
		Checkouts: []Checkout{{ComputerID: created.ID, HarnessProjectID: "t3-proj-1"}},
	})
	require.Equal(t, http.StatusOK, rec.Code)
	var link ProjectLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &link))

	rec = doRequest(routes, http.MethodGet, "/api/pairing/projects", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"links":[{"project_id":"proj-1","provider":"claude","model":"sonnet","placement":"spread","computers":["`+created.ID+
		`"],"checkouts":[{"computer_id":"`+created.ID+`","harness_project_id":"t3-proj-1"}],"updated_at":"`+link.UpdatedAt.Format(time.RFC3339Nano)+`"}]}`, rec.Body.String())

	rec = doRequest(routes, http.MethodDelete, "/api/pairing/projects/proj-1", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(routes, http.MethodGet, "/api/pairing/projects/proj-1", "u1", nil)
	var afterClear ProjectLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &afterClear))
	assert.Empty(t, afterClear.Computers, "clearing goes back to the person's order")
	assert.Equal(t, []Checkout{{ComputerID: created.ID, HarnessProjectID: "t3-proj-1"}}, afterClear.Checkouts, "where the code is stays known")
}

func TestHandler_ProjectLink_RefusesComputersThatAreNotTheirsOrListedTwice(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()
	theirs := pairedComputer(t, h, "u2", "Their box")
	mine := pairedComputer(t, h, "u1", "Home")

	tests := map[string]struct {
		req  projectLinkRequest
		want int
	}{
		"someone else's computer in the list":    {projectLinkRequest{Computers: []string{theirs.ID}}, http.StatusNotFound},
		"someone else's computer as a checkout":  {projectLinkRequest{Checkouts: []Checkout{{ComputerID: theirs.ID, HarnessProjectID: "p"}}}, http.StatusNotFound},
		"a computer listed twice":                {projectLinkRequest{Computers: []string{mine.ID, mine.ID}}, http.StatusBadRequest},
		"a checkout without its T3 project":      {projectLinkRequest{Checkouts: []Checkout{{ComputerID: mine.ID}}}, http.StatusBadRequest},
		"a placement that is neither of the two": {projectLinkRequest{Placement: "random"}, http.StatusBadRequest},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rec := doRequest(routes, http.MethodPut, "/api/pairing/projects/proj-1", "u1", tt.req)
			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestHandler_Computers_OrderAndRunsAtOnce(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()
	home := pairedComputer(t, h, "u1", "Home")
	vps := pairedComputer(t, h, "u1", "VPS")

	rec := doRequest(routes, http.MethodPut, "/api/pairing/computers/order", "u1", reorderRequest{ComputerIDs: []string{vps.ID, home.ID}})
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doRequest(routes, http.MethodGet, "/api/pairing/computers", "u1", nil)
	var listed struct {
		Computers []Computer `json:"computers"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed.Computers, 2)
	assert.Equal(t, []string{vps.ID, home.ID}, []string{listed.Computers[0].ID, listed.Computers[1].ID})

	rec = doRequest(routes, http.MethodPut, "/api/pairing/computers/order", "u1", reorderRequest{ComputerIDs: []string{vps.ID}})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "an order that leaves a computer out is refused")

	rec = doRequest(routes, http.MethodPatch, "/api/pairing/computers/"+home.ID, "u1", map[string]any{"runs_at_once": 3, "chat_project_id": "t3-chat"})
	require.Equal(t, http.StatusOK, rec.Code)
	var patched Computer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &patched))
	assert.Equal(t, []any{"Home", 3, "t3-chat"}, []any{patched.Name, patched.RunsAtOnce, patched.ChatProjectID}, "an omitted name keeps its value")

	for _, n := range []int{0, MaxRunsAtOnce + 1} {
		rec = doRequest(routes, http.MethodPatch, "/api/pairing/computers/"+home.ID, "u1", map[string]any{"runs_at_once": n})
		assert.Equal(t, http.StatusBadRequest, rec.Code, "runs at once %d", n)
	}
	rec = doRequest(routes, http.MethodPatch, "/api/pairing/computers/"+home.ID, "u2", map[string]any{"runs_at_once": 2})
	assert.Equal(t, http.StatusNotFound, rec.Code, "only the owner sets it")
}

func TestHandler_Resolve_Success(t *testing.T) {
	t.Parallel()
	h, _, _ := newTestHandler()
	routes := h.Routes()

	created := pairedComputer(t, h, "u1", "Home")
	rec := doRequest(routes, http.MethodPut, "/api/pairing/defaults", "u1", defaultsRequest{Provider: "claude", Model: "sonnet"})
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doRequest(routes, http.MethodPatch, "/api/pairing/computers/"+created.ID, "u1", map[string]any{"chat_project_id": "harness-proj"})
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
	assert.Equal(t, []any{}, body["passed_over"])
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

	t.Run("no computer can take it", func(t *testing.T) {
		t.Parallel()
		h, _, _ := newTestHandler()
		routes := h.Routes()
		home := pairedComputer(t, h, "u1", "Home")

		rec := doRequest(routes, http.MethodGet, "/api/pairing/resolve", "u1", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"ok":false,"reason":"no_computer_ready","passed_over":[{"computer_id":"`+home.ID+
			`","name":"Home","why":"no_chat_folder","text":"Home has no chat folder"}]}`, rec.Body.String())
	})

	t.Run("expired_token", func(t *testing.T) {
		t.Parallel()
		repo := newFakeRepo()
		exch := &fakeExchanger{result: harness.PairResult{BearerToken: "b", ExpiresIn: time.Hour}, version: "0.0.34"}
		svc := newTestService(repo, exch)
		routes := NewHandler(svc).Routes()
		created := pairedComputer(t, NewHandler(svc), "u1", "Home")
		rec := doRequest(routes, http.MethodPatch, "/api/pairing/computers/"+created.ID, "u1", map[string]any{"chat_project_id": "p"})
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
	rec := doRequest(routes, http.MethodPatch, "/api/pairing/computers/"+created.ID, "u1", map[string]any{"chat_project_id": "p"})
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

func TestHandler_ProjectsListSaysWhetherTheComputerCanClone(t *testing.T) {
	t.Parallel()
	h, _, exch := newTestHandler()
	exch.CanCloneFn = func(context.Context, harness.Session) (bool, error) { return false, nil }
	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(h.Routes(), http.MethodGet, "/api/pairing/computers/"+created.ID+"/projects", "u1", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"projects":[],"can_clone":false}`, rec.Body.String())
}

func TestHandler_ProjectsPost_MakesAnEmptyProjectFromAName(t *testing.T) {
	t.Parallel()
	h, _, exch := newTestHandler()
	var made []string
	exch.CreateProjectFn = func(_ context.Context, _ harness.Session, title, path string) (harness.Project, error) {
		made = append(made, title+" at "+path)
		return harness.Project{ID: "t3-new", Title: title, Path: path}, nil
	}
	created := pairedComputer(t, h, "u1", "Home")

	rec := doRequest(h.Routes(), http.MethodPost, "/api/pairing/computers/"+created.ID+"/projects", "u1", map[string]string{"name": "notes"})
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.JSONEq(t, `{"id":"t3-new","title":"notes","path":"~/Code/notes"}`, rec.Body.String())
	assert.Equal(t, []string{"notes at ~/Code/notes"}, made)

	rec = doRequest(h.Routes(), http.MethodPost, "/api/pairing/computers/"+created.ID+"/projects", "u1", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "neither a repository nor a name")
}
