package memories

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

func (f *fakeRepo) findAnswer(projectID string, round int, question string) int {
	for i, a := range f.answers {
		if a.ProjectID == projectID && a.Round == round && a.Question == question {
			return i
		}
	}
	return -1
}

func (f *fakeRepo) ListAnswers(_ context.Context, projectID string) ([]*InterviewAnswer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answerErr != nil {
		return nil, f.answerErr
	}
	out := []*InterviewAnswer{}
	for _, a := range f.answers {
		if a.ProjectID == projectID {
			cp := *a
			out = append(out, &cp)
		}
	}
	slices.SortStableFunc(out, func(a, b *InterviewAnswer) int { return a.Round - b.Round })
	return out, nil
}

func (f *fakeRepo) UpsertAnswer(_ context.Context, a *InterviewAnswer, evts ...eventbus.OutboxEvent) (*InterviewAnswer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answerErr != nil {
		return nil, f.answerErr
	}
	cp := *a
	if i := f.findAnswer(a.ProjectID, a.Round, a.Question); i >= 0 {
		cp.ID = f.answers[i].ID
		f.answers[i] = &cp
	}
	if f.findAnswer(a.ProjectID, a.Round, a.Question) < 0 {
		f.answers = append(f.answers, &cp)
	}
	f.events = append(f.events, evts...)
	out := cp
	return &out, nil
}

func (f *fakeRepo) UpdateAnswer(_ context.Context, a *InterviewAnswer, evts ...eventbus.OutboxEvent) (*InterviewAnswer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answerErr != nil {
		return nil, f.answerErr
	}
	i := f.findAnswer(a.ProjectID, a.Round, a.Question)
	if i < 0 {
		return nil, apperrs.ErrNotFound
	}
	stored := f.answers[i]
	stored.Selected, stored.Text, stored.Skipped, stored.AnsweredBy, stored.AnsweredAt = a.Selected, a.Text, a.Skipped, a.AnsweredBy, a.AnsweredAt
	f.events = append(f.events, evts...)
	out := *stored
	return &out, nil
}

func (f *fakeRepo) DeleteAnswer(_ context.Context, projectID string, round int, question string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answerErr != nil {
		return f.answerErr
	}
	i := f.findAnswer(projectID, round, question)
	if i < 0 {
		return apperrs.ErrNotFound
	}
	f.answers = append(f.answers[:i], f.answers[i+1:]...)
	f.events = append(f.events, evts...)
	return nil
}

func (f *fakeRepo) LastRound(_ context.Context, projectID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answerErr != nil {
		return 0, f.answerErr
	}
	last := 0
	for _, a := range f.answers {
		if a.ProjectID == projectID {
			last = max(last, a.Round)
		}
	}
	return last, nil
}

func (f *fakeRepo) InsertRound(_ context.Context, answers []*InterviewAnswer, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range answers {
		cp := *a
		f.answers = append(f.answers, &cp)
	}
	f.events = append(f.events, evts...)
	return nil
}

func answer(question string, selected ...string) InterviewAnswer {
	return InterviewAnswer{ProjectID: "project-1", Question: question, Selected: selected}
}

func TestSaveAnswer_Errors(t *testing.T) {
	repoErr := errors.New("disk full")
	tests := []struct {
		name    string
		access  fakeAccess
		ctx     context.Context
		answer  InterviewAnswer
		repoErr error
		wantErr error
	}{
		{"no memories:write", fakeAccess{can: true, deny: map[string]bool{"project-1:" + string(permissions.MemoriesWrite): true}}, testCtx(), answer("Testing", "Unit"), nil, apperrs.ErrForbidden},
		{"no project access", fakeAccess{can: true, outside: map[string]bool{"project-1": true}}, testCtx(), answer("Testing", "Unit"), nil, apperrs.ErrNotFound},
		{"unknown project", fakeAccess{can: true}, testCtx(), InterviewAnswer{ProjectID: "project-9", Question: "Testing", Text: "x"}, nil, apperrs.ErrNotFound},
		{"no actor", fakeAccess{can: true}, context.Background(), answer("Testing", "Unit"), nil, apperrs.ErrUnauthorized},
		{"no project", fakeAccess{can: true}, testCtx(), InterviewAnswer{Question: "Testing", Text: "x"}, nil, apperrs.ErrInvalid},
		{"empty question", fakeAccess{can: true}, testCtx(), answer("  ", "Unit"), nil, apperrs.ErrInvalid},
		{"question over the cap", fakeAccess{can: true}, testCtx(), answer(strings.Repeat("q", maxQuestionChars+1), "Unit"), nil, apperrs.ErrInvalid},
		{"negative round", fakeAccess{can: true}, testCtx(), InterviewAnswer{ProjectID: "project-1", Round: -1, Question: "Testing", Text: "x"}, nil, apperrs.ErrInvalid},
		{"nothing picked or written", fakeAccess{can: true}, testCtx(), answer("Testing", " "), nil, apperrs.ErrInvalid},
		{"too many picks", fakeAccess{can: true}, testCtx(), answer("Testing", strings.Split(strings.Repeat("x,", maxSelected+1), ",")...), nil, apperrs.ErrInvalid},
		{"text over the cap", fakeAccess{can: true}, testCtx(), InterviewAnswer{ProjectID: "project-1", Question: "Testing", Text: strings.Repeat("t", MaxInterviewChars+1)}, nil, apperrs.ErrInvalid},
		{"follow-up never asked", fakeAccess{can: true}, testCtx(), InterviewAnswer{ProjectID: "project-1", Round: 1, Question: "Coverage?", Text: "80"}, nil, apperrs.ErrNotFound},
		{"repo failure", fakeAccess{can: true}, testCtx(), answer("Testing", "Unit"), repoErr, repoErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.answerErr = tt.repoErr
			_, err := newTestServiceWith(repo, tt.access).SaveAnswer(tt.ctx, tt.answer)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.answers)
			assert.Empty(t, repo.eventsFor(TopicAnswerSaved))
		})
	}
}

func TestSaveAnswer_ReAnswerReplaces_AndSkipDropsTheAnswer(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)

	first, err := s.SaveAnswer(testCtx(), answer(" Testing ", " Unit ", "", "E2E"))
	require.NoError(t, err)
	assert.Equal(t, "Testing", first.Question)
	assert.Equal(t, []string{"Unit", "E2E"}, first.Selected)
	assert.Equal(t, "workspace-1", first.WorkspaceID)
	assert.Equal(t, "user-1", first.AnsweredBy)

	again, err := s.SaveAnswer(testCtx(), InterviewAnswer{ProjectID: "project-1", Question: "Testing", Text: "Integration only"})
	require.NoError(t, err)
	assert.Equal(t, first.ID, again.ID, "re-answering keeps the row")
	assert.Empty(t, again.Selected)
	assert.Equal(t, "Integration only", again.Text)

	skipped, err := s.SaveAnswer(testCtx(), InterviewAnswer{ProjectID: "project-1", Question: "Testing", Selected: []string{"Unit"}, Text: "x", Skipped: true})
	require.NoError(t, err)
	assert.True(t, skipped.Skipped)
	assert.Empty(t, skipped.Selected)
	assert.Empty(t, skipped.Text)

	as, err := s.ListAnswers(testCtx(), "project-1")
	require.NoError(t, err)
	require.Len(t, as, 1)
	evts := repo.eventsFor(TopicAnswerSaved)
	require.Len(t, evts, 3)
	assert.Equal(t, AnswerEvent{WorkspaceID: "workspace-1", ProjectID: "project-1", Question: "Testing", AuthorID: "user-1", At: fixedNow}, evts[2].Payload)
}

func TestClearAnswer(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.SaveAnswer(testCtx(), answer("Testing", "Unit"))
	require.NoError(t, err)
	_, err = s.RecordRound(testCtx(), "project-1", "user-1", []InterviewAnswer{{Question: "Coverage floor?", Options: []AnswerOption{{Label: "80%"}}, Why: "CI enforces none", Selected: []string{"80%"}}})
	require.NoError(t, err)

	require.NoError(t, s.ClearAnswer(testCtx(), "project-1", 0, "Testing"))
	require.NoError(t, s.ClearAnswer(testCtx(), "project-1", 0, "Testing"), "clearing an unanswered question does nothing")
	require.NoError(t, s.ClearAnswer(testCtx(), "project-1", 1, "Coverage floor?"))

	as, err := s.ListAnswers(testCtx(), "project-1")
	require.NoError(t, err)
	require.Len(t, as, 1, "the template question's row is gone")
	assert.Equal(t, "Coverage floor?", as[0].Question, "a follow-up keeps its question")
	assert.Equal(t, "CI enforces none", as[0].Why)
	assert.Empty(t, as[0].Selected)
	assert.False(t, as[0].Skipped)
	assert.Len(t, repo.eventsFor(TopicAnswerCleared), 2)

	tests := []struct {
		name     string
		access   fakeAccess
		ctx      context.Context
		round    int
		question string
		wantErr  error
	}{
		{"no memories:write", fakeAccess{can: true, deny: map[string]bool{"project-1:" + string(permissions.MemoriesWrite): true}}, testCtx(), 0, "Testing", apperrs.ErrForbidden},
		{"no actor", fakeAccess{can: true}, context.Background(), 0, "Testing", apperrs.ErrUnauthorized},
		{"empty question", fakeAccess{can: true}, testCtx(), 0, " ", apperrs.ErrInvalid},
		{"negative round", fakeAccess{can: true}, testCtx(), -2, "Testing", apperrs.ErrInvalid},
		{"follow-up never asked", fakeAccess{can: true}, testCtx(), 4, "Testing", apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newTestServiceWith(repo, tt.access).ClearAnswer(tt.ctx, "project-1", tt.round, tt.question)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
	t.Run("repo failure", func(t *testing.T) {
		failing := newFakeRepo()
		failing.answerErr = errors.New("disk full")
		require.ErrorIs(t, newTestService(failing).ClearAnswer(testCtx(), "project-1", 0, "Testing"), failing.answerErr)
	})
}

func TestListAnswers_Errors(t *testing.T) {
	repoErr := errors.New("disk full")
	tests := []struct {
		name      string
		access    fakeAccess
		projectID string
		repoErr   error
		wantErr   error
	}{
		{"no project", fakeAccess{can: true}, " ", nil, apperrs.ErrInvalid},
		{"no memories:read", fakeAccess{can: false}, "project-1", nil, apperrs.ErrForbidden},
		{"no project access", fakeAccess{can: true, outside: map[string]bool{"project-1": true}}, "project-1", nil, apperrs.ErrNotFound},
		{"unknown project", fakeAccess{can: true}, "project-9", nil, apperrs.ErrNotFound},
		{"repo failure", fakeAccess{can: true}, "project-1", repoErr, repoErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.answerErr = tt.repoErr
			_, err := newTestServiceWith(repo, tt.access).ListAnswers(testCtx(), tt.projectID)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestRecordRound_NumbersRoundsAndKeepsTheFollowUps(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := s.SaveAnswer(testCtx(), answer("Testing", "Unit"))
	require.NoError(t, err)

	round, err := s.RecordRound(context.Background(), "project-1", "user-2", []InterviewAnswer{
		{Question: "Coverage floor?", Options: []AnswerOption{{Label: "80%", Description: "the CI gate"}, {Label: "None"}}, Why: "No gate in CI", Selected: []string{"80%"}},
		{Question: "Mocks?", Options: []AnswerOption{{Label: "Fakes"}}, MultiSelect: true, Why: "Both styles appear"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, round)
	round, err = s.RecordRound(context.Background(), "project-1", "user-2", []InterviewAnswer{{Question: "Coverage floor?", Text: "90"}})
	require.NoError(t, err)
	assert.Equal(t, 2, round, "the same question in a later round is a new row")

	as, err := s.ListAnswers(testCtx(), "project-1")
	require.NoError(t, err)
	require.Len(t, as, 4)
	assert.Equal(t, []int{0, 1, 1, 2}, []int{as[0].Round, as[1].Round, as[2].Round, as[3].Round})
	assert.Equal(t, "No gate in CI", as[1].Why)
	assert.Equal(t, "user-2", as[1].AnsweredBy)
	assert.True(t, as[2].Skipped, "a follow-up with no answer is a skip")
	assert.True(t, as[2].MultiSelect)

	updated, err := s.SaveAnswer(testCtx(), InterviewAnswer{ProjectID: "project-1", Round: 1, Question: "Coverage floor?", Selected: []string{"None"}})
	require.NoError(t, err)
	assert.Equal(t, []string{"None"}, updated.Selected)
	assert.Equal(t, "No gate in CI", updated.Why, "re-answering a follow-up keeps what was asked")
	assert.Len(t, repo.eventsFor(TopicAnswerSaved), 5)
}

func TestRecordRound_Errors(t *testing.T) {
	repoErr := errors.New("disk full")
	tests := []struct {
		name      string
		projectID string
		followUps []InterviewAnswer
		repoErr   error
		wantErr   error
	}{
		{"no project", "", []InterviewAnswer{{Question: "Q", Text: "a"}}, nil, apperrs.ErrInvalid},
		{"no follow-ups", "project-1", nil, nil, apperrs.ErrInvalid},
		{"unknown project", "project-9", []InterviewAnswer{{Question: "Q", Text: "a"}}, nil, apperrs.ErrNotFound},
		{"empty question", "project-1", []InterviewAnswer{{Question: " ", Text: "a"}}, nil, apperrs.ErrInvalid},
		{"asked twice", "project-1", []InterviewAnswer{{Question: "Q", Text: "a"}, {Question: " Q ", Text: "b"}}, nil, apperrs.ErrInvalid},
		{"repo failure", "project-1", []InterviewAnswer{{Question: "Q", Text: "a"}}, repoErr, repoErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.answerErr = tt.repoErr
			_, err := newTestService(repo).RecordRound(context.Background(), tt.projectID, "user-1", tt.followUps)
			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, repo.answers)
		})
	}
}

func TestAnswerHandlers(t *testing.T) {
	h, repo := newMemoriesHandler()

	rec := serve(t, h, http.MethodPut, "/api/memories/interview-answers", `{"project_id":"project-1","question":"Testing","selected":["Unit"]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	var saved InterviewAnswer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
	assert.Equal(t, []string{"Unit"}, saved.Selected)

	rec = serve(t, h, http.MethodPost, "/api/memories/interview-answers/skip", `{"project_id":"project-1","question":"Stack"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"skipped":true`)

	rec = serve(t, h, http.MethodGet, "/api/memories/interview-answers?project_id=project-1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var listed []InterviewAnswer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Len(t, listed, 2)

	rec = serve(t, h, http.MethodPost, "/api/memories/interview-answers/clear", `{"project_id":"project-1","question":"Stack"}`)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Len(t, repo.answers, 1)

	for _, bad := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/memories/interview-answers", ""},
		{http.MethodPut, "/api/memories/interview-answers", `{`},
		{http.MethodPut, "/api/memories/interview-answers", `{"project_id":"project-1","question":""}`},
		{http.MethodPost, "/api/memories/interview-answers/clear", `{`},
		{http.MethodPost, "/api/memories/interview-answers/clear", `{"project_id":"project-1","round":-1,"question":"Q"}`},
	} {
		rec = serve(t, h, bad.method, bad.path, bad.body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "%s %s %s", bad.method, bad.path, bad.body)
	}
	deny := NewHandler(newDenyService(newFakeRepo())).Routes()
	rec = serve(t, deny, http.MethodPost, "/api/memories/interview-answers/skip", `{"project_id":"project-1","question":"Stack"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestMemoryTools_InterviewAnswers(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	interview, err := s.CreateInterview(testCtx(), "project-1", "")
	require.NoError(t, err)
	ordinary := mustMemory(t, s, "project-1", "Tone", "when", "body", false)
	id := `"id":"` + interview.ID + `"`

	out := mustCall(t, s, "memory_get", `{`+id+`}`).(memoryResult)
	assert.NotNil(t, out.Answers, "the interview memory always carries its answers, none yet")
	assert.Empty(t, out.Answers)

	out = mustCall(t, s, "memory_update", `{`+id+`,"answers":[{"question":"Testing","selected":["Unit"]},{"question":"Stack","skip":true}]}`).(memoryResult)
	require.Len(t, out.Answers, 2)
	assert.Equal(t, interview.Version, out.Version, "answers alone add no version")
	assert.True(t, out.Answers[1].Skipped)

	created := mustCall(t, s, "memory_create", `{"project_id":"project-1","kind":"interview"}`).(memoryResult)
	assert.Len(t, created.Answers, 2)

	plain := mustCall(t, s, "memory_get", `{"id":"`+ordinary.ID+`"}`).(memoryResult)
	assert.Nil(t, plain.Answers)

	tests := []struct {
		name    string
		svc     *Service
		args    string
		wantErr error
	}{
		{"answers on an ordinary memory", s, `{"id":"` + ordinary.ID + `","answers":[{"question":"Testing","text":"x"}]}`, apperrs.ErrInvalid},
		{"an empty answer", s, `{` + id + `,"answers":[{"question":"Testing"}]}`, apperrs.ErrInvalid},
		{"answers with a revert", s, `{` + id + `,"revert_to_version":1,"answers":[{"question":"Testing","text":"x"}]}`, apperrs.ErrInvalid},
		{"answers without memories:write", newTestServiceWith(repo, fakeAccess{can: true, deny: map[string]bool{"project-1:" + string(permissions.MemoriesWrite): true}}), `{` + id + `,"answers":[{"question":"Testing","text":"x"}]}`, apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(testCtx(), t, tt.svc, "memory_update", tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
	t.Run("answers unreadable", func(t *testing.T) {
		repo.answerErr = errors.New("disk full")
		t.Cleanup(func() { repo.answerErr = nil })
		_, err := callTool(testCtx(), t, s, "memory_get", `{`+id+`}`)
		require.ErrorIs(t, err, repo.answerErr)
	})
}
