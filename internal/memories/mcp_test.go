package memories

import (
	"context"
	"encoding/json"
	"strings"
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

func mustCall(t *testing.T, s *Service, name, args string) any {
	t.Helper()
	out, err := callTool(testCtx(), t, s, name, args)
	require.NoError(t, err)
	return out
}

func mustMemory(t *testing.T, s *Service, projectID, workspaceID, title, whenToUse, body string, always bool) *Memory {
	t.Helper()
	m, err := s.Create(testCtx(), projectID, workspaceID, title, whenToUse, body, always, "")
	require.NoError(t, err)
	return m
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(newTestService(newFakeRepo())) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
	}
	assert.Equal(t, []string{
		"memory_list", "memory_get", "memory_create", "memory_update", "memory_delete",
		"interview_template_get", "interview_template_update",
	}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	repo := newFakeRepo()
	allowed, denied := newTestService(repo), newDenyService(repo)
	m := mustMemory(t, allowed, "project-1", "", "Title", "when", "body", false)
	interview, err := allowed.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)
	id := `"id":"` + m.ID + `"`
	tests := []struct {
		name    string
		svc     *Service
		ctx     context.Context
		tool    string
		args    string
		wantErr error
	}{
		{"get without id", allowed, testCtx(), "memory_get", `{}`, apperrs.ErrInvalid},
		{"get a non-positive version", allowed, testCtx(), "memory_get", `{` + id + `,"version":-1}`, apperrs.ErrInvalid},
		{"list with neither project nor workspace", allowed, testCtx(), "memory_list", `{}`, apperrs.ErrInvalid},
		{"create an ordinary memory without a title", allowed, testCtx(), "memory_create", `{"project_id":"project-1"}`, apperrs.ErrInvalid},
		{"create at workspace scope without a workspace", allowed, testCtx(), "memory_create", `{"title":"Tone"}`, apperrs.ErrInvalid},
		{"create an unknown kind", allowed, testCtx(), "memory_create", `{"project_id":"project-1","title":"x","kind":"notes"}`, apperrs.ErrInvalid},
		{"clone with new content", allowed, testCtx(), "memory_create", `{"clone_from_id":"` + m.ID + `","project_id":"project-2","title":"x"}`, apperrs.ErrInvalid},
		{"clone with nowhere to go", allowed, testCtx(), "memory_create", `{"clone_from_id":"` + m.ID + `"}`, apperrs.ErrInvalid},
		{"interview with a body it would drop", allowed, testCtx(), "memory_create", `{"project_id":"project-1","kind":"interview","body":"## Stack"}`, apperrs.ErrInvalid},
		{"update with an unknown key", allowed, testCtx(), "memory_update", `{` + id + `,"version":2}`, apperrs.ErrInvalid},
		{"update to a blank title", allowed, testCtx(), "memory_update", `{` + id + `,"title":" "}`, apperrs.ErrInvalid},
		{"revert mixed with a field", allowed, testCtx(), "memory_update", `{` + id + `,"revert_to_version":1,"title":"x"}`, apperrs.ErrInvalid},
		{"interview body over the cap", allowed, testCtx(), "memory_update", `{"id":"` + interview.ID + `","body":"` + strings.Repeat("a", MaxInterviewChars+1) + `"}`, apperrs.ErrInvalid},
		{"template update without a workspace", allowed, testCtx(), "interview_template_update", `{"body":"x"}`, apperrs.ErrInvalid},
		{"get a missing memory", allowed, testCtx(), "memory_get", `{"id":"nope"}`, apperrs.ErrNotFound},
		{"get a missing version", allowed, testCtx(), "memory_get", `{` + id + `,"version":9}`, apperrs.ErrNotFound},
		{"update a missing memory", allowed, testCtx(), "memory_update", `{"id":"nope","title":"x"}`, apperrs.ErrNotFound},
		{"revert to a missing version", allowed, testCtx(), "memory_update", `{` + id + `,"revert_to_version":9}`, apperrs.ErrNotFound},
		{"delete a missing memory", allowed, testCtx(), "memory_delete", `{"id":"nope"}`, apperrs.ErrNotFound},
		{"interview of a missing project", allowed, testCtx(), "memory_create", `{"project_id":"nope","kind":"interview"}`, apperrs.ErrNotFound},
		{"list without read", denied, testCtx(), "memory_list", `{"project_id":"project-1"}`, apperrs.ErrForbidden},
		{"get without read", denied, testCtx(), "memory_get", `{` + id + `}`, apperrs.ErrForbidden},
		{"create without write", denied, testCtx(), "memory_create", `{"project_id":"project-1","title":"x"}`, apperrs.ErrForbidden},
		{"update without access", denied, testCtx(), "memory_update", `{` + id + `,"title":"x"}`, apperrs.ErrForbidden},
		{"delete without delete", denied, testCtx(), "memory_delete", `{` + id + `}`, apperrs.ErrForbidden},
		{"template update without write", denied, testCtx(), "interview_template_update", `{"workspace_id":"workspace-1","body":"x"}`, apperrs.ErrForbidden},
		{"create without an actor", allowed, context.Background(), "memory_create", `{"project_id":"project-1","title":"x"}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(tt.ctx, t, tt.svc, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestMemoryUpdate_OmittedFieldsKeepTheirValues(t *testing.T) {
	s := newTestService(newFakeRepo())
	m := mustMemory(t, s, "project-1", "", "Title", "when deploying", "**v1**", true)

	got := mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","title":"Renamed"}`).(memoryResult)
	assert.Equal(t, "Renamed", got.Title)
	assert.Equal(t, "when deploying", got.WhenToUse)
	assert.Equal(t, "**v1**", got.Body)
	assert.True(t, got.AlwaysIncluded)
	assert.Equal(t, 2, got.Version)

	got = mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","always_included":false}`).(memoryResult)
	assert.False(t, got.AlwaysIncluded)
	assert.Equal(t, "Renamed", got.Title)
	assert.Equal(t, "**v1**", got.Body)

	got = mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","when_to_use":""}`).(memoryResult)
	assert.Empty(t, got.WhenToUse, "an empty string clears the when-to-use line")
	assert.Equal(t, "**v1**", got.Body)

	got = mustCall(t, s, "memory_update", `{"id":"`+m.ID+`"}`).(memoryResult)
	assert.Equal(t, 4, got.Version, "an empty update saves no version")
}

func TestMemoryUpdate_RevertToVersion(t *testing.T) {
	s := newTestService(newFakeRepo())
	m := mustMemory(t, s, "project-1", "", "Title", "when", "v1", false)
	mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","body":"v2"}`)

	got := mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","revert_to_version":1}`).(memoryResult)
	assert.Equal(t, "v1", got.Body)
	assert.Equal(t, 3, got.Version)
	v, err := s.GetVersion(testCtx(), m.ID, 3)
	require.NoError(t, err)
	assert.Equal(t, viaMCP, v.AuthorVia)
}

func TestInterviewTemplateUpdate_OmittedBodyKeepsTheTemplate(t *testing.T) {
	s := newTestService(newFakeRepo())
	mustCall(t, s, "interview_template_update", `{"workspace_id":"workspace-1","body":"## Mine"}`)

	got := mustCall(t, s, "interview_template_update", `{"workspace_id":"workspace-1"}`).(*InterviewTemplate)
	assert.Equal(t, "## Mine", got.Body)
	got = mustCall(t, s, "interview_template_get", `{"workspace_id":"workspace-1"}`).(*InterviewTemplate)
	assert.Equal(t, "## Mine", got.Body)
	assert.Equal(t, DefaultInterviewTemplate, got.DefaultBody)
}

func TestMemoryCreate(t *testing.T) {
	s := newTestService(newFakeRepo())

	t.Run("an ordinary project memory", func(t *testing.T) {
		got := mustCall(t, s, "memory_create", `{"project_id":"project-1","title":"Deploy quirks","when_to_use":"when deploying","body":"**bold**","always_included":true}`).(memoryResult)
		assert.Equal(t, "project-1", got.ProjectID)
		assert.Equal(t, "**bold**", got.Body)
		assert.True(t, got.AlwaysIncluded)
		v, err := s.GetVersion(testCtx(), got.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, viaMCP, v.AuthorVia)
	})
	t.Run("a workspace memory", func(t *testing.T) {
		got := mustCall(t, s, "memory_create", `{"workspace_id":"workspace-1","title":"Team tone"}`).(memoryResult)
		assert.Empty(t, got.ProjectID)
		assert.Equal(t, "workspace-1", got.WorkspaceID)
	})
	t.Run("the decisions log, once per project", func(t *testing.T) {
		got := mustCall(t, s, "memory_create", `{"project_id":"project-2","kind":"decisions_log","body":"entry","always_included":true}`).(memoryResult)
		assert.Equal(t, KindDecisionsLog, got.Kind)
		assert.False(t, got.AlwaysIncluded)
		_, err := callTool(testCtx(), t, s, "memory_create", `{"project_id":"project-2","kind":"decisions_log"}`)
		require.ErrorIs(t, err, apperrs.ErrConflict)
	})
	t.Run("the interview memory is get-or-create", func(t *testing.T) {
		first := mustCall(t, s, "memory_create", `{"project_id":"project-1","kind":"interview"}`).(memoryResult)
		assert.Equal(t, KindInterview, first.Kind)
		assert.Contains(t, first.Body, "Stack and versions")
		again := mustCall(t, s, "memory_create", `{"project_id":"project-1","kind":"interview"}`).(memoryResult)
		assert.Equal(t, first.ID, again.ID)
	})
	t.Run("a clone into another project", func(t *testing.T) {
		src := mustMemory(t, s, "project-1", "", "Source", "when", "body", false)
		got := mustCall(t, s, "memory_create", `{"clone_from_id":"`+src.ID+`","project_id":"project-2"}`).(memoryResult)
		assert.NotEqual(t, src.ID, got.ID)
		assert.Equal(t, "project-2", got.ProjectID)
		assert.Equal(t, "Source", got.Title)
	})
}

func TestMemoryGet(t *testing.T) {
	s := newTestService(newFakeRepo())
	m := mustMemory(t, s, "project-1", "", "Title", "when", "v1", false)
	mustCall(t, s, "memory_update", `{"id":"`+m.ID+`","title":"Renamed","body":"# v2"}`)

	got := mustCall(t, s, "memory_get", `{"id":"`+m.ID+`"}`).(memoryResult)
	assert.Equal(t, "# v2", got.Body)
	assert.Equal(t, 2, got.Version)
	assert.Zero(t, got.CurrentVersion)
	require.Len(t, got.Versions, 2)
	assert.Equal(t, 2, got.Versions[0].Version, "newest first")
	assert.Equal(t, viaMCP, got.Versions[0].AuthorVia)

	old := mustCall(t, s, "memory_get", `{"id":"`+m.ID+`","version":1}`).(memoryResult)
	assert.Equal(t, "v1", old.Body)
	assert.Equal(t, "Title", old.Title)
	assert.Equal(t, 1, old.Version)
	assert.Equal(t, 2, old.CurrentVersion)
}

func TestVersionInfos_KeepsTheNewest(t *testing.T) {
	vs := make([]*MemoryVersion, memoryGetVersions+5)
	for i := range vs {
		vs[i] = &MemoryVersion{Version: len(vs) - i}
	}
	got := versionInfos(vs)
	require.Len(t, got, memoryGetVersions)
	assert.Equal(t, len(vs), got[0].Version)
}

func TestMemoryList(t *testing.T) {
	s := newTestService(newFakeRepo())
	mustMemory(t, s, "project-1", "", "Project note", "when", "secret body", false)
	mustMemory(t, s, "", "workspace-1", "Team tone", "always", "body", true)

	page := func(t *testing.T, args string) mcptool.Page[memoryListItem] {
		t.Helper()
		return mustCall(t, s, "memory_list", args).(mcptool.Page[memoryListItem])
	}
	t.Run("a project's list includes its workspace memories", func(t *testing.T) {
		assert.Equal(t, 2, page(t, `{"project_id":"project-1"}`).Total)
	})
	t.Run("a workspace's list has only workspace memories", func(t *testing.T) {
		p := page(t, `{"workspace_id":"workspace-1"}`)
		require.Len(t, p.Items, 1)
		assert.Equal(t, "Team tone", p.Items[0].Title)
		assert.True(t, p.Items[0].AlwaysIncluded)
	})
	t.Run("items carry no body", func(t *testing.T) {
		b, err := json.Marshal(page(t, `{"project_id":"project-1"}`))
		require.NoError(t, err)
		assert.NotContains(t, string(b), "secret body")
	})
	t.Run("limit pages the list", func(t *testing.T) {
		p := page(t, `{"project_id":"project-1","limit":1}`)
		assert.Len(t, p.Items, 1)
		assert.Equal(t, 1, p.NextOffset)
	})
}

func TestMemoryDelete_ReturnsWhatItDeleted(t *testing.T) {
	s := newTestService(newFakeRepo())
	m := mustMemory(t, s, "project-1", "", "Title", "when", "body", false)
	assert.Equal(t, mcptool.Gone(m.ID), mustCall(t, s, "memory_delete", `{"id":"`+m.ID+`"}`))
	_, err := s.Get(testCtx(), m.ID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}
