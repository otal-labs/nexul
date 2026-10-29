package storage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

type historyFixture struct {
	svc *docs.Service
	doc *docs.Doc
}

// newHistoryFixture files one doc by alice in a workspace where bob is not a member.
func newHistoryFixture(t *testing.T) historyFixture {
	t.Helper()
	s := storage.New(newDB(t), []byte("0123456789abcdef0123456789abcdef"))
	accessSvc := access.NewService(s.Access, realUsers{s.Users})
	svc := docs.NewService(s.Docs, accessSvc, nil)
	seedUser(t, s, "alice", "alice")
	seedUser(t, s, "bob", "bob")
	joinDefaultWorkspace(t, s, accessSvc, "alice")
	doc, err := svc.Create(actor("alice"), "project-general", "Spec", "first")
	require.NoError(t, err)
	return historyFixture{svc: svc, doc: doc}
}

func (f historyFixture) versionNumbers(t *testing.T) []int {
	t.Helper()
	hist, err := f.svc.History(actor("alice"), f.doc.ID)
	require.NoError(t, err)
	out := []int{}
	for _, v := range hist.Versions {
		out = append(out, v.Version)
	}
	return out
}

func TestDocUpdate_UnchangedTitleAndBody_WritesNoVersion(t *testing.T) {
	f := newHistoryFixture(t)
	ctx := actor("alice")

	_, err := f.svc.Update(ctx, f.doc.ID, "Spec", "first")
	require.NoError(t, err)
	assert.Len(t, f.versionNumbers(t), 1, "a save that changes nothing adds no version")

	_, err = f.svc.Update(ctx, f.doc.ID, "Spec renamed", "first")
	require.NoError(t, err)
	assert.Len(t, f.versionNumbers(t), 2, "a title-only change is a change")

	_, err = f.svc.Update(ctx, f.doc.ID, "Spec renamed", "first")
	require.NoError(t, err)
	assert.Len(t, f.versionNumbers(t), 2, "saving what the newest version already holds adds none")
}

func TestDocUpdate_PastTenUnnamedVersions_PrunesTheOldestAndKeepsNamedOnes(t *testing.T) {
	f := newHistoryFixture(t)
	ctx := actor("alice")
	_, err := f.svc.CreateNamedVersion(ctx, f.doc.ID, "Milestone")
	require.NoError(t, err)

	for i := range 9 {
		_, err = f.svc.Update(ctx, f.doc.ID, "Spec", fmt.Sprintf("edit %d", i))
		require.NoError(t, err)
	}
	assert.Contains(t, f.versionNumbers(t), 1, "ten unnamed versions are all kept")

	_, err = f.svc.Update(ctx, f.doc.ID, "Spec", "edit 9")
	require.NoError(t, err)
	got := f.versionNumbers(t)
	assert.NotContains(t, got, 1, "the eleventh unnamed version prunes the oldest")
	assert.Contains(t, got, 2, "the named version survives the prune")
	assert.Len(t, got, 11, "ten unnamed plus the named one")
}

func callDocTool(t *testing.T, svc *docs.Service, name, input string) map[string]any {
	t.Helper()
	for _, tl := range docs.MCPTools(svc) {
		if tl.Name != name {
			continue
		}
		out, err := tl.Call(actor("alice"), json.RawMessage(input))
		require.NoError(t, err)
		raw, err := json.Marshal(out)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		return m
	}
	t.Fatalf("no tool %s", name)
	return nil
}

func TestDocSave_EditorsAndVia_RecordedFromAUserAnAutomationAndAnMCPCall(t *testing.T) {
	f := newHistoryFixture(t)
	byAutomation := identity.WithActor(context.Background(), identity.Actor{
		ID: "alice", Automation: &identity.AutomationRef{ID: "auto-1", Name: "Nightly sync"},
	})
	_, err := f.svc.Update(actor("alice"), f.doc.ID, "Spec", "by hand")
	require.NoError(t, err)
	_, err = f.svc.Update(byAutomation, f.doc.ID, "Spec", "by automation")
	require.NoError(t, err)
	callDocTool(t, f.svc, "doc_update", fmt.Sprintf(`{"id":%q,"body":"by agent"}`, f.doc.ID))
	created := callDocTool(t, f.svc, "doc_create", `{"project_id":"project-general","title":"Agent doc","body":"hello"}`)

	hist, err := f.svc.History(actor("alice"), f.doc.ID)
	require.NoError(t, err)
	require.Len(t, hist.Versions, 4)
	for i, wantVia := range []string{"mcp", "Nightly sync", "", ""} {
		assert.Equal(t, wantVia, hist.Versions[i].Via, "version %d", hist.Versions[i].Version)
		assert.Equal(t, []string{"alice"}, hist.Versions[i].EditorIDs)
	}

	createdHist, err := f.svc.History(actor("alice"), created["id"].(string))
	require.NoError(t, err)
	require.Len(t, createdHist.Versions, 1)
	assert.Equal(t, "mcp", createdHist.Versions[0].Via, "a create records how it came")

	saved, err := f.svc.GetVersion(actor("alice"), f.doc.ID, hist.Versions[2].Version)
	require.NoError(t, err)
	assert.Equal(t, "alice", saved.AuthorID, "the author is the first editor")
}

func getHistory(t *testing.T, f historyFixture, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/docs/"+f.doc.ID+"/history", nil).WithContext(actor(user))
	rec := httptest.NewRecorder()
	docs.NewHandler(f.svc).Routes().ServeHTTP(rec, req)
	return rec
}

func TestDocHistoryRoute_AReader_GetsSummariesNewestFirstWithoutBodies(t *testing.T) {
	f := newHistoryFixture(t)
	_, err := f.svc.Update(actor("alice"), f.doc.ID, "Spec", "second")
	require.NoError(t, err)
	_, err = f.svc.CreateNamedVersion(actor("alice"), f.doc.ID, "Milestone")
	require.NoError(t, err)

	rec := getHistory(t, f, "alice")
	require.Equal(t, http.StatusOK, rec.Code)
	var raw struct {
		Versions []map[string]any `json:"versions"`
		Changes  []any            `json:"changes"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	assert.Equal(t, []any{}, raw.Changes, "changes is an empty array, not null")
	require.Len(t, raw.Versions, 3)
	for _, v := range raw.Versions {
		assert.ElementsMatch(t, []string{"version", "created_at", "name", "editor_ids", "via", "current"}, keysOf(v), "no body or title")
	}
	assert.Equal(t, []float64{3, 2, 1}, []float64{raw.Versions[0]["version"].(float64), raw.Versions[1]["version"].(float64), raw.Versions[2]["version"].(float64)})
	assert.Equal(t, "Milestone", raw.Versions[0]["name"])
	assert.Equal(t, []bool{true, false, false}, []bool{raw.Versions[0]["current"].(bool), raw.Versions[1]["current"].(bool), raw.Versions[2]["current"].(bool)})
}

func TestDocHistoryRoute_AUserWithoutDocsRead_Answers404(t *testing.T) {
	f := newHistoryFixture(t)

	assert.Equal(t, http.StatusNotFound, getHistory(t, f, "bob").Code, "no access reads as no such doc")
	req := httptest.NewRequest(http.MethodGet, "/api/docs/missing/history", nil).WithContext(actor("alice"))
	rec := httptest.NewRecorder()
	docs.NewHandler(f.svc).Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code, "a doc that does not exist answers the same")
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
