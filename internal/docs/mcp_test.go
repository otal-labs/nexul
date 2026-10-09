package docs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func callTool(ctx context.Context, t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func mustDoc(t *testing.T, s *Service, projectID, title, body string) *Doc {
	t.Helper()
	d, err := s.Create(testCtx(), projectID, title, body)
	require.NoError(t, err)
	return d
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(newFakeRepo())) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
		assert.NotNil(t, tool.InputSchema, tool.Name)
	}
	assert.Equal(t, []string{"doc_list", "doc_get", "doc_create", "doc_update", "doc_delete"}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	repo := newFakeRepo()
	d := mustDoc(t, newTestService(repo), "project-1", "Spec", "body")
	allowed, denied := newTestService(repo), newDenyService(repo)
	tests := []struct {
		name    string
		svc     *Service
		ctx     context.Context
		tool    string
		args    string
		wantErr error
	}{
		{"get without id", allowed, testCtx(), "doc_get", `{}`, apperrs.ErrInvalid},
		{"get with an unknown key", allowed, testCtx(), "doc_get", `{"id":"` + d.ID + `","body":"x"}`, apperrs.ErrInvalid},
		{"create without a project", allowed, testCtx(), "doc_create", `{"title":"Spec"}`, apperrs.ErrInvalid},
		{"create with a blank title", allowed, testCtx(), "doc_create", `{"project_id":"project-1","title":"  "}`, apperrs.ErrInvalid},
		{"update with a non-boolean archived", allowed, testCtx(), "doc_update", `{"id":"` + d.ID + `","archived":"yes"}`, apperrs.ErrInvalid},
		{"update to a blank title", allowed, testCtx(), "doc_update", `{"id":"` + d.ID + `","title":" "}`, apperrs.ErrInvalid},
		{"list with a blank query", allowed, testCtx(), "doc_list", `{"query":" "}`, apperrs.ErrInvalid},
		{"get a missing doc", allowed, testCtx(), "doc_get", `{"id":"nope"}`, apperrs.ErrNotFound},
		{"update a missing doc", allowed, testCtx(), "doc_update", `{"id":"nope","title":"x"}`, apperrs.ErrNotFound},
		{"get without read", denied, testCtx(), "doc_get", `{"id":"` + d.ID + `"}`, apperrs.ErrForbidden},
		{"update without read or write", denied, testCtx(), "doc_update", `{"id":"` + d.ID + `","title":"x"}`, apperrs.ErrForbidden},
		{"watch without read", denied, testCtx(), "doc_update", `{"id":"` + d.ID + `","watch":true}`, apperrs.ErrForbidden},
		{"delete without id", allowed, testCtx(), "doc_delete", `{}`, apperrs.ErrInvalid},
		{"delete a missing doc", allowed, testCtx(), "doc_delete", `{"id":"nope"}`, apperrs.ErrNotFound},
		{"delete without the delete bit", denied, testCtx(), "doc_delete", `{"id":"` + d.ID + `"}`, apperrs.ErrForbidden},
		{"create without an actor", allowed, context.Background(), "doc_create", `{"project_id":"project-1","title":"Spec"}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(tt.ctx, t, tt.svc, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDocDelete_RemovesTheDocRatherThanArchivingIt(t *testing.T) {
	s := newTestService(newFakeRepo())
	d := mustDoc(t, s, "project-1", "Research", "body")

	out, err := callTool(testCtx(), t, s, "doc_delete", `{"id":"`+d.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone(d.ID), out)

	_, err = callTool(testCtx(), t, s, "doc_get", `{"id":"`+d.ID+`"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	listed, err := callTool(testCtx(), t, s, "doc_list", `{"include_archived":true}`)
	require.NoError(t, err)
	assert.Empty(t, listed.(mcptool.Page[*DocListItem]).Items)
}

func TestDocUpdate_OmittedFieldsKeepTheirValues(t *testing.T) {
	s := newTestService(newFakeRepo())
	d := mustDoc(t, s, "project-1", "Spec", "# Body")

	out, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","title":"Renamed"}`)
	require.NoError(t, err)
	got := out.(docResult)
	assert.Equal(t, "Renamed", got.Title)
	assert.Equal(t, "# Body", got.Body, "a title-only update keeps the body")
	assert.Equal(t, 2, got.Version)

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","body":"**v3**"}`)
	require.NoError(t, err)
	got = out.(docResult)
	assert.Equal(t, "Renamed", got.Title, "a body-only update keeps the title")
	assert.Equal(t, "**v3**", got.Body)

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","archived":true}`)
	require.NoError(t, err)
	got = out.(docResult)
	assert.True(t, got.Archived)
	assert.Equal(t, "**v3**", got.Body, "archiving keeps the body")
	assert.Equal(t, 3, got.Version, "archiving saves no version")

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","archived":false}`)
	require.NoError(t, err)
	assert.False(t, out.(docResult).Archived, "archived false restores")

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, 3, out.(docResult).Version, "an empty update saves nothing")
}

func TestDocUpdate_LockOrdersAroundTheEdit(t *testing.T) {
	s := newTestService(newFakeRepo())
	d := mustDoc(t, s, "project-1", "Spec", "body")

	out, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","title":"Final","locked":true}`)
	require.NoError(t, err, "the title saves before the lock lands")
	got := out.(docResult)
	assert.Equal(t, "Final", got.Title)
	assert.True(t, got.Locked)

	_, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","body":"changed"}`)
	require.ErrorIs(t, err, apperrs.ErrConflict, "a locked doc refuses a body")

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","body":"changed","locked":false}`)
	require.NoError(t, err, "the unlock lands before the body saves")
	got = out.(docResult)
	assert.Equal(t, "changed", got.Body)
	assert.False(t, got.Locked)
}

func TestDocUpdate_WatchStartsAndStopsForTheCallerOnly(t *testing.T) {
	s := newTestService(newFakeRepo())
	d := mustDoc(t, s, "project-1", "Spec", "body")
	other := identity.WithActor(context.Background(), identity.Actor{ID: "user-2"})

	out, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","watch":true}`)
	require.NoError(t, err)
	got := out.(docResult)
	assert.True(t, got.Watching)
	assert.Equal(t, []watcherResult{{UserID: "user-1", Source: WatcherManual}}, got.Watchers)
	assert.Equal(t, 1, got.Version, "watching is not an edit")

	out, err = callTool(other, t, s, "doc_get", `{"id":"`+d.ID+`"}`)
	require.NoError(t, err)
	assert.False(t, out.(docResult).Watching, "watching is the caller's own, never someone else's")
	assert.Len(t, out.(docResult).Watchers, 1)

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","title":"Renamed"}`)
	require.NoError(t, err)
	assert.True(t, out.(docResult).Watching, "an update without watch leaves it as is")

	out, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","watch":false}`)
	require.NoError(t, err)
	assert.False(t, out.(docResult).Watching)
	assert.Empty(t, out.(docResult).Watchers)
}

func TestStepErr_SaysWhatWasSaved(t *testing.T) {
	assert.Equal(t, apperrs.ErrForbidden, stepErr(nil, "archived", apperrs.ErrForbidden))
	err := stepErr([]string{"title"}, "archived", apperrs.ErrForbidden)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "already applied: title")
	var partial *mcptool.PartialError
	assert.ErrorAs(t, err, &partial, "the adapter shows what was saved even when it hides the failure")
}

func TestDocGetAndCreate_SpeakMarkdown(t *testing.T) {
	s := newTestService(newFakeRepo())
	out, err := callTool(testCtx(), t, s, "doc_create", `{"project_id":"project-1","title":"Spec","body":"# Title\n\nBody **bold**"}`)
	require.NoError(t, err)
	created := out.(docResult)
	assert.Equal(t, "project-1", created.ProjectID)
	assert.Equal(t, 1, created.Version)

	out, err = callTool(testCtx(), t, s, "doc_get", `{"id":"`+created.ID+`"}`)
	require.NoError(t, err)
	assert.Equal(t, "# Title\n\nBody **bold**", out.(docResult).Body)
}

func TestDocList(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	spine := mustDoc(t, s, "project-1", "Storage spine", "SQLite migrations")
	mustDoc(t, s, "project-2", "SQLite tuning", "pragmas")
	page := func(t *testing.T, s *Service, args string) mcptool.Page[*DocListItem] {
		t.Helper()
		out, err := callTool(testCtx(), t, s, "doc_list", args)
		require.NoError(t, err)
		return out.(mcptool.Page[*DocListItem])
	}

	t.Run("limit pages the result", func(t *testing.T) {
		p := page(t, s, `{"limit":1}`)
		assert.Len(t, p.Items, 1)
		assert.Equal(t, 2, p.Total)
		assert.True(t, p.HasMore)
	})
	t.Run("a doc the caller cannot open is listed without access", func(t *testing.T) {
		p := page(t, newDenyService(repo), `{"project_id":"project-1"}`)
		require.Len(t, p.Items, 1)
		assert.False(t, p.Items[0].CanOpen)
		assert.Empty(t, p.Items[0].Snippet)
	})
	t.Run("a search leaves out what the caller cannot read, and counts only the rest", func(t *testing.T) {
		readsSpine := NewService(repo, fakeAccess{readable: []string{spine.ID}}, nil)
		p := page(t, readsSpine, `{"query":"sqlite"}`)
		require.Len(t, p.Items, 1)
		assert.Equal(t, spine.ID, p.Items[0].ID)
		assert.True(t, p.Items[0].CanOpen)
		assert.Equal(t, 1, p.Total)
	})
}

func TestDocTools_Folders(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	getSource, err := s.CreateFolder(testCtx(), "project-1", "GetSource")
	require.NoError(t, err)
	stays := mustDoc(t, s, "project-1", "Roadmap", "")

	got, err := callTool(testCtx(), t, s, "doc_create", `{"project_id":"project-1","folder_id":"`+getSource.ID+`","title":"EP01"}`)
	require.NoError(t, err)
	ep01 := got.(docResult)
	assert.Equal(t, getSource.ID, ep01.FolderID)

	got, err = callTool(testCtx(), t, s, "doc_update", `{"id":"`+stays.ID+`","folder_id":"`+getSource.ID+`","title":"Roadmap v2"}`)
	require.NoError(t, err)
	moved := got.(docResult)
	assert.Equal(t, getSource.ID, moved.FolderID, "folder_id moves the doc")
	assert.Equal(t, "Roadmap v2", moved.Title, "alongside the other fields sent")

	got, err = callTool(testCtx(), t, s, "doc_list", `{"folder_id":"main-project-1"}`)
	require.NoError(t, err)
	assert.Empty(t, got.(mcptool.Page[*DocListItem]).Items, "both docs left Main")

	_, err = callTool(testCtx(), t, s, "doc_create", `{"project_id":"project-1","clone_from_id":"`+ep01.ID+`","folder_id":"`+getSource.ID+`"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a copy's folder is not chosen at copy time")
}
