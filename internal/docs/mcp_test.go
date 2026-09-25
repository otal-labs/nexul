package docs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
	assert.Equal(t, []string{"doc_list", "doc_get", "doc_create", "doc_update"}, names)
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
		{"create without an actor", allowed, context.Background(), "doc_create", `{"project_id":"project-1","title":"Spec"}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(tt.ctx, t, tt.svc, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
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

func TestArchiveErr_SaysWhatWasSaved(t *testing.T) {
	assert.Equal(t, apperrs.ErrForbidden, archiveErr(false, apperrs.ErrForbidden))
	err := archiveErr(true, apperrs.ErrForbidden)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "the title and body were saved")
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
	old := mustDoc(t, s, "project-1", "Old plan", "retired")
	_, err := s.Archive(testCtx(), old.ID)
	require.NoError(t, err)

	page := func(t *testing.T, args string) mcptool.Page[*DocListItem] {
		t.Helper()
		out, err := callTool(testCtx(), t, s, "doc_list", args)
		require.NoError(t, err)
		return out.(mcptool.Page[*DocListItem])
	}
	ids := func(p mcptool.Page[*DocListItem]) []string {
		var out []string
		for _, d := range p.Items {
			out = append(out, d.ID)
		}
		return out
	}

	t.Run("browsing hides archived docs", func(t *testing.T) {
		assert.Equal(t, 2, page(t, `{}`).Total)
	})
	t.Run("include_archived lists them too", func(t *testing.T) {
		assert.Equal(t, 3, page(t, `{"include_archived":true}`).Total)
	})
	t.Run("project_id narrows the list", func(t *testing.T) {
		assert.Equal(t, []string{spine.ID}, ids(page(t, `{"project_id":"project-1"}`)))
	})
	t.Run("a query searches across projects", func(t *testing.T) {
		assert.Equal(t, 2, page(t, `{"query":"sqlite"}`).Total)
	})
	t.Run("a query within a project", func(t *testing.T) {
		assert.Equal(t, []string{spine.ID}, ids(page(t, `{"query":"sqlite","project_id":"project-1"}`)))
	})
	t.Run("limit pages the result", func(t *testing.T) {
		p := page(t, `{"limit":1}`)
		assert.Len(t, p.Items, 1)
		assert.True(t, p.HasMore)
	})
	t.Run("a doc the caller cannot open is listed without access", func(t *testing.T) {
		out, err := callTool(testCtx(), t, newDenyService(repo), "doc_list", `{"project_id":"project-1"}`)
		require.NoError(t, err)
		p := out.(mcptool.Page[*DocListItem])
		require.Len(t, p.Items, 1)
		assert.False(t, p.Items[0].CanOpen)
	})
}
