package memories

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func (f *fakeRepo) GetByProjectKind(_ context.Context, projectID, kind string) (*Memory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, m := range f.memories {
		if m.ProjectID == projectID && m.Kind == kind {
			cp := *m
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) GetInterviewTemplate(_ context.Context, workspaceID string) (*InterviewTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.templateErr != nil {
		return nil, f.templateErr
	}
	t, ok := f.templates[workspaceID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (f *fakeRepo) SaveInterviewTemplate(_ context.Context, t *InterviewTemplate, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.templateErr != nil {
		return f.templateErr
	}
	if f.templates == nil {
		f.templates = map[string]*InterviewTemplate{}
	}
	cp := *t
	f.templates[t.WorkspaceID] = &cp
	f.events = append(f.events, evts...)
	return nil
}

func markdownOf(t *testing.T, body string) string {
	t.Helper()
	md, err := richtext.ToMarkdown(body)
	require.NoError(t, err)
	return md
}

func TestCreateInterview_NoInterviewYet_CopiesTheDefaultTemplate(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)

	m, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)
	assert.Equal(t, KindInterview, m.Kind)
	assert.True(t, m.AlwaysIncluded)
	assert.Equal(t, "workspace-1", m.WorkspaceID)
	assert.Contains(t, markdownOf(t, m.Body), "Stack and versions")
	require.Len(t, repo.eventsFor(TopicCreated), 1)
}

func TestCreateInterview_ExistingInterview_ReturnsItUnchanged(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	first, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)

	again, err := s.CreateInterview(testCtx(), "project-1", "mcp")
	require.NoError(t, err)
	assert.Equal(t, first.ID, again.ID)
	assert.Len(t, repo.eventsFor(TopicCreated), 1)
}

func TestCreateInterview_StartsFromTheSavedTemplate_AndLaterTemplateEditsLeaveItAlone(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.SaveInterviewTemplate(testCtx(), "workspace-1", "## Our stack\nGo only.")
	require.NoError(t, err)

	m, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)
	assert.Contains(t, markdownOf(t, m.Body), "Go only.")

	_, err = s.SaveInterviewTemplate(testCtx(), "workspace-1", "## Changed")
	require.NoError(t, err)
	got, err := s.Get(testCtx(), m.ID)
	require.NoError(t, err)
	assert.Contains(t, markdownOf(t, got.Body), "Go only.")
}

func TestCreateInterview_Errors(t *testing.T) {
	tests := []struct {
		name    string
		svc     func() *Service
		ctx     context.Context
		project string
		want    error
	}{
		{"empty project", func() *Service { return newTestService(newFakeRepo()) }, testCtx(), " ", apperrs.ErrInvalid},
		{"no actor", func() *Service { return newTestService(newFakeRepo()) }, context.Background(), "project-1", apperrs.ErrUnauthorized},
		{"unknown project", func() *Service { return newTestService(newFakeRepo()) }, testCtx(), "nope", apperrs.ErrNotFound},
		{"no write permission", func() *Service { return newDenyService(newFakeRepo()) }, testCtx(), "project-1", apperrs.ErrForbidden},
		{"lookup fails", func() *Service {
			repo := newFakeRepo()
			repo.getErr = errors.New("disk")
			return newTestService(repo)
		}, testCtx(), "project-1", nil},
		{"template read fails", func() *Service {
			repo := newFakeRepo()
			repo.templateErr = errors.New("disk")
			return newTestService(repo)
		}, testCtx(), "project-1", nil},
		{"create fails", func() *Service {
			repo := newFakeRepo()
			repo.createErr = errors.New("disk")
			return newTestService(repo)
		}, testCtx(), "project-1", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc().CreateInterview(tt.ctx, tt.project, "")
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestUpdate_InterviewMemory_StaysAlwaysIncluded(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)

	got, err := s.Update(testCtx(), m.ID, "Interview", "", "## Stack\nGo.", false, "mcp")
	require.NoError(t, err)
	assert.True(t, got.AlwaysIncluded)
	assert.Equal(t, KindInterview, got.Kind)
}

func TestUpdate_InterviewMemoryOverTheCap_IsRefusedWithTheCount(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)

	_, err = s.Update(testCtx(), m.ID, "Interview", "", strings.Repeat("a", MaxInterviewChars+1), true, "")
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "8001 characters, over the 8000-character cap")
}

func TestUpdate_OrdinaryMemoryOverTheInterviewCap_IsAllowed(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.Create(testCtx(), "project-1", "", "Notes", "", "x", false, "")
	require.NoError(t, err)

	got, err := s.Update(testCtx(), m.ID, "Notes", "", strings.Repeat("a", MaxInterviewChars+1), false, "")
	require.NoError(t, err)
	assert.False(t, got.AlwaysIncluded)
}

func TestRevert_InterviewMemory_WorksLikeAnyMemory(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)
	_, err = s.Update(testCtx(), m.ID, "Interview", "", "## Stack\nRust.", true, "")
	require.NoError(t, err)

	got, err := s.Revert(testCtx(), m.ID, 1, "")
	require.NoError(t, err)
	assert.Equal(t, 3, got.Version)
	assert.Contains(t, markdownOf(t, got.Body), "Stack and versions")
}

func TestClone_InterviewMemory_BecomesAnOrdinaryMemory(t *testing.T) {
	s := newTestService(newFakeRepo())
	m, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)

	clone, err := s.Clone(testCtx(), m.ID, "project-2", "")
	require.NoError(t, err)
	assert.Empty(t, clone.Kind)
}

func TestListMemoryItems_CarriesTheKind(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)

	_, projectItems, err := s.ListMemoryItems(context.Background(), "workspace-1", "project-1")
	require.NoError(t, err)
	require.Len(t, projectItems, 1)
	assert.Equal(t, KindInterview, projectItems[0].Kind)
	assert.NotEmpty(t, projectItems[0].Body)
}

func TestInterviewTemplate_NeverSaved_ReturnsTheDefault(t *testing.T) {
	s := newTestService(newFakeRepo())
	got, err := s.InterviewTemplate(testCtx(), "workspace-1")
	require.NoError(t, err)
	assert.Equal(t, DefaultInterviewTemplate, got.Body)
	assert.Equal(t, DefaultInterviewTemplate, got.DefaultBody)
}

func TestSaveInterviewTemplate_PublishesAndReadsBack(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.SaveInterviewTemplate(testCtx(), "workspace-1", "## Only this")
	require.NoError(t, err)

	got, err := s.InterviewTemplate(testCtx(), "workspace-1")
	require.NoError(t, err)
	assert.Equal(t, "## Only this", got.Body)
	assert.Equal(t, DefaultInterviewTemplate, got.DefaultBody)
	require.Len(t, repo.eventsFor(TopicInterviewTemplateUpdated), 1)
}

func TestInterviewTemplate_Errors(t *testing.T) {
	failing := newFakeRepo()
	failing.templateErr = errors.New("disk")
	tests := []struct {
		name string
		call func() error
		want error
	}{
		{"get: empty workspace", func() error {
			_, err := newTestService(newFakeRepo()).InterviewTemplate(testCtx(), " ")
			return err
		}, apperrs.ErrInvalid},
		{"get: no read permission", func() error {
			_, err := newDenyService(newFakeRepo()).InterviewTemplate(testCtx(), "workspace-1")
			return err
		}, apperrs.ErrForbidden},
		{"get: repo fails", func() error {
			_, err := newTestService(failing).InterviewTemplate(testCtx(), "workspace-1")
			return err
		}, nil},
		{"save: empty workspace", func() error {
			_, err := newTestService(newFakeRepo()).SaveInterviewTemplate(testCtx(), "", "x")
			return err
		}, apperrs.ErrInvalid},
		{"save: no actor", func() error {
			_, err := newTestService(newFakeRepo()).SaveInterviewTemplate(context.Background(), "workspace-1", "x")
			return err
		}, apperrs.ErrUnauthorized},
		{"save: no write permission", func() error {
			_, err := newDenyService(newFakeRepo()).SaveInterviewTemplate(testCtx(), "workspace-1", "x")
			return err
		}, apperrs.ErrForbidden},
		{"save: over the cap", func() error {
			_, err := newTestService(newFakeRepo()).SaveInterviewTemplate(testCtx(), "workspace-1", strings.Repeat("a", MaxInterviewChars+1))
			return err
		}, apperrs.ErrInvalid},
		{"save: repo fails", func() error {
			_, err := newTestService(failing).SaveInterviewTemplate(testCtx(), "workspace-1", "x")
			return err
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			require.Error(t, err)
			if tt.want != nil {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestInterviewHandlers(t *testing.T) {
	h, _ := newMemoriesHandler()

	rec := serve(t, h, http.MethodPost, "/api/memories/interview", `{"project_id":"project-1"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, KindInterview, decodeMemory(t, rec).Kind)

	rec = serve(t, h, http.MethodPost, "/api/memories/interview", `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	rec = serve(t, h, http.MethodPost, "/api/memories/interview", `{`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, h, http.MethodGet, "/api/memories/interview-template?workspace_id=workspace-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Stack and versions")
	rec = serve(t, h, http.MethodGet, "/api/memories/interview-template", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, h, http.MethodPut, "/api/memories/interview-template", `{"workspace_id":"workspace-1","body":"## Mine"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "## Mine")
	rec = serve(t, h, http.MethodPut, "/api/memories/interview-template", `{"workspace_id":""}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	rec = serve(t, h, http.MethodPut, "/api/memories/interview-template", `{`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestInterviewMCPTools(t *testing.T) {
	tools := MCPTools(newTestService(newFakeRepo()))

	got, err := toolByName(t, tools, "memory_create_interview").Call(testCtx(), map[string]any{"project_id": "project-1"})
	require.NoError(t, err)
	m, ok := got.(*Memory)
	require.True(t, ok)
	assert.Contains(t, m.Body, "Stack and versions")
	_, err = toolByName(t, tools, "memory_create_interview").Call(testCtx(), map[string]any{})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = toolByName(t, tools, "memory_create_interview").Call(testCtx(), map[string]any{"project_id": "nope"})
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	_, err = toolByName(t, tools, "memory_update").Call(testCtx(), map[string]any{"id": m.ID, "title": "Interview", "body": strings.Repeat("a", MaxInterviewChars+1)})
	require.ErrorIs(t, err, apperrs.ErrInvalid)

	_, err = toolByName(t, tools, "interview_template_update").Call(testCtx(), map[string]any{"workspace_id": "workspace-1", "body": "## Mine"})
	require.NoError(t, err)
	_, err = toolByName(t, tools, "interview_template_update").Call(testCtx(), map[string]any{})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	tmpl, err := toolByName(t, tools, "interview_template_get").Call(testCtx(), map[string]any{"workspace_id": "workspace-1"})
	require.NoError(t, err)
	assert.Equal(t, "## Mine", tmpl.(*InterviewTemplate).Body)
	_, err = toolByName(t, tools, "interview_template_get").Call(testCtx(), map[string]any{})
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
