package tickets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// fakeStages maps status ids to stages; FirstOfStage returns the first id listed for a stage.
type fakeStages struct {
	stages   map[string]string
	order    []string
	stageErr error
	firstErr error
}

func (f fakeStages) StageOf(_ context.Context, statusID string) (string, error) {
	if f.stageErr != nil {
		return "", f.stageErr
	}
	stage, ok := f.stages[statusID]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return stage, nil
}

func (f fakeStages) FirstOfStage(_ context.Context, _, stage string) (Status, error) {
	if f.firstErr != nil {
		return "", f.firstErr
	}
	for _, id := range f.order {
		if f.stages[id] == stage {
			return Status(id), nil
		}
	}
	return "", apperrs.ErrNotFound
}

type threadPost struct {
	ticketID, authorID, body string
}

type fakeThreads struct {
	posts []threadPost
	err   error
}

func (f *fakeThreads) PostToTicketThread(_ context.Context, t *Ticket, authorID, body string) error {
	if f.err != nil {
		return f.err
	}
	f.posts = append(f.posts, threadPost{ticketID: t.ID, authorID: authorID, body: body})
	return nil
}

type fakeTargets struct {
	got    []BranchLink
	target TestTarget
	err    error
}

func (f *fakeTargets) ResolveTestTarget(_ context.Context, _ string, branches []BranchLink) (TestTarget, error) {
	f.got = branches
	return f.target, f.err
}

type testingFixture struct {
	svc     *Service
	repo    *fakeRepo
	threads *fakeThreads
	targets *fakeTargets
}

func newTestingFixture(status Status) testingFixture {
	repo := newFakeRepo()
	repo.tickets["t1"] = &Ticket{ID: "t1", ProjectID: "p1", Title: "Login", Status: status}
	s := NewService(repo, fakeStatusStore{known: map[Status]bool{"build": true, "qa": true, "shipped": true}}, fakeUserLogins{logins: map[string]string{"u-1": "onik97"}})
	threads, targets := &fakeThreads{}, &fakeTargets{}
	s.SetTesting(Testing{
		Stages:  fakeStages{stages: map[string]string{"build": StageProgress, "qa": "testing", "shipped": StageDone}, order: []string{"build", "qa", "shipped"}},
		Threads: threads,
		Targets: targets,
	})
	return testingFixture{svc: s, repo: repo, threads: threads, targets: targets}
}

func asUser(ctx context.Context) context.Context {
	return identity.WithActor(ctx, identity.Actor{ID: "u-1"})
}

func eventOf(t *testing.T, evts []eventbus.OutboxEvent, topic string) eventbus.OutboxEvent {
	t.Helper()
	for _, e := range evts {
		if e.Topic == topic {
			return e
		}
	}
	require.Failf(t, "event not published", "topic %s", topic)
	return eventbus.OutboxEvent{}
}

func TestTestPass_MovesToFirstDoneColumnAndRecordsTester(t *testing.T) {
	f := newTestingFixture("qa")

	got, err := f.svc.TestPass(asUser(t.Context()), "t1", false)

	require.NoError(t, err)
	assert.Equal(t, Status("shipped"), got.Status)
	assert.Equal(t, "onik97", got.Tester)
	evt := eventOf(t, f.repo.events, TopicTestPassed).Payload.(TestedEvent)
	assert.Equal(t, "onik97", evt.Tester)
	assert.Equal(t, Status("shipped"), evt.Ticket.Status)
	eventOf(t, f.repo.events, TopicTesterChanged)
	eventOf(t, f.repo.events, TopicStatusChanged)
	assert.Equal(t, []threadPost{{ticketID: "t1", authorID: "u-1", body: "Passed by onik97"}}, f.threads.posts)
}

func TestTestPass_KeepsAssignedTesterAndNamesTheTarget(t *testing.T) {
	f := newTestingFixture("qa")
	f.repo.tickets["t1"].Tester = "qa-lead"
	f.targets.target = TestTarget{URL: "https://login.example.com", Kind: "preview"}

	got, err := f.svc.TestPass(asUser(t.Context()), "t1", false)

	require.NoError(t, err)
	assert.Equal(t, "qa-lead", got.Tester)
	assert.Equal(t, "onik97", eventOf(t, f.repo.events, TopicTestPassed).Payload.(TestedEvent).Tester)
	assert.Equal(t, "Passed by onik97 on https://login.example.com", f.threads.posts[0].body)
	for _, e := range f.repo.events {
		assert.NotEqual(t, TopicTesterChanged, e.Topic)
	}
}

func TestTestPass_Errors(t *testing.T) {
	t.Run("no signed-in tester", func(t *testing.T) {
		_, err := newTestingFixture("qa").svc.TestPass(t.Context(), "t1", false)
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
	})
	t.Run("missing ticket", func(t *testing.T) {
		_, err := newTestingFixture("qa").svc.TestPass(asUser(t.Context()), "nope", false)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("project without a done column", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.svc.testing.Stages = fakeStages{stages: map[string]string{"qa": "testing"}, order: []string{"qa"}}
		_, err := f.svc.TestPass(asUser(t.Context()), "t1", false)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.ErrorContains(t, err, "no done-stage column")
	})
	t.Run("column lookup fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.svc.testing.Stages = fakeStages{firstErr: errors.New("db down")}
		_, err := f.svc.TestPass(asUser(t.Context()), "t1", false)
		assert.ErrorContains(t, err, "db down")
	})
	t.Run("test target lookup fails before moving", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.targets.err = errors.New("stacks down")
		_, err := f.svc.TestPass(asUser(t.Context()), "t1", false)
		assert.ErrorContains(t, err, "stacks down")
		assert.Equal(t, Status("qa"), f.repo.tickets["t1"].Status)
	})
	t.Run("move fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.repo.statusErr = errors.New("write failed")
		_, err := f.svc.TestPass(asUser(t.Context()), "t1", false)
		assert.ErrorContains(t, err, "write failed")
		assert.Empty(t, f.threads.posts)
	})
	t.Run("posting fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.threads.err = errors.New("chat down")
		_, err := f.svc.TestPass(asUser(t.Context()), "t1", false)
		assert.ErrorContains(t, err, "chat down")
	})
}

func TestTestFail_PostsReportAndMovesBackToProgress(t *testing.T) {
	f := newTestingFixture("qa")
	report := TestReport{Steps: "Open /login", Expected: "A form", Actual: "A blank page", Screenshots: []string{"att-1"}}

	got, err := f.svc.TestFail(asUser(t.Context()), "t1", report, false)

	require.NoError(t, err)
	assert.Equal(t, Status("build"), got.Status)
	require.Len(t, f.threads.posts, 1)
	post := f.threads.posts[0]
	assert.Equal(t, "u-1", post.authorID)
	assert.Equal(t, "t1", post.ticketID)
	want := "Test failed\n\n## Steps to reproduce\nOpen /login\n\n## Expected result\nA form\n\n## Actual result\nA blank page\n\n## Screenshot\n![screenshot](/api/attachments/att-1)"
	assert.Equal(t, want, post.body)
	evt := eventOf(t, f.repo.events, TopicTestFailed).Payload.(TestedEvent)
	assert.Equal(t, want, evt.Report)
	assert.Equal(t, "onik97", evt.Tester)
	assert.Equal(t, Status("build"), evt.Ticket.Status)
}

func TestTestFail_LeavesOutEmptySections(t *testing.T) {
	f := newTestingFixture("qa")

	_, err := f.svc.TestFail(asUser(t.Context()), "t1", TestReport{Actual: "  crashes  "}, false)

	require.NoError(t, err)
	assert.Equal(t, "Test failed\n\n## Actual result\ncrashes", f.threads.posts[0].body)
}

func TestTestFail_Errors(t *testing.T) {
	report := TestReport{Actual: "broken"}
	t.Run("no actual result", func(t *testing.T) {
		_, err := newTestingFixture("qa").svc.TestFail(asUser(t.Context()), "t1", TestReport{Steps: "x"}, false)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("screenshot that is not an attachment id", func(t *testing.T) {
		_, err := newTestingFixture("qa").svc.TestFail(asUser(t.Context()), "t1", TestReport{Actual: "x", Screenshots: []string{"a)\n![x](https://evil)"}}, false)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
	})
	t.Run("done ticket is never reopened", func(t *testing.T) {
		f := newTestingFixture("shipped")
		_, err := f.svc.TestFail(asUser(t.Context()), "t1", report, false)
		assert.ErrorIs(t, err, apperrs.ErrInvalid)
		assert.ErrorContains(t, err, "never reopened")
		assert.Equal(t, Status("shipped"), f.repo.tickets["t1"].Status)
		assert.Empty(t, f.threads.posts)
	})
	t.Run("stage lookup fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.svc.testing.Stages = fakeStages{stageErr: errors.New("db down")}
		_, err := f.svc.TestFail(asUser(t.Context()), "t1", report, false)
		assert.ErrorContains(t, err, "db down")
	})
	t.Run("missing ticket", func(t *testing.T) {
		_, err := newTestingFixture("qa").svc.TestFail(asUser(t.Context()), "nope", report, false)
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("project without a progress column", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.svc.testing.Stages = fakeStages{stages: map[string]string{"qa": "testing"}, order: []string{"qa"}}
		_, err := f.svc.TestFail(asUser(t.Context()), "t1", report, false)
		assert.ErrorContains(t, err, "no progress-stage column")
	})
	t.Run("no signed-in tester", func(t *testing.T) {
		f := newTestingFixture("qa")
		_, err := f.svc.TestFail(t.Context(), "t1", report, false)
		assert.ErrorIs(t, err, apperrs.ErrUnauthorized)
		assert.Equal(t, Status("qa"), f.repo.tickets["t1"].Status)
	})
	t.Run("move fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.repo.statusErr = errors.New("write failed")
		_, err := f.svc.TestFail(asUser(t.Context()), "t1", report, false)
		assert.ErrorContains(t, err, "write failed")
		assert.Empty(t, f.threads.posts)
	})
	t.Run("posting fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.threads.err = errors.New("chat down")
		_, err := f.svc.TestFail(asUser(t.Context()), "t1", report, false)
		assert.ErrorContains(t, err, "chat down")
	})
}

func TestTestResult_SignsByPath(t *testing.T) {
	pass := func(viaMCP bool) func(context.Context, *Service) error {
		return func(ctx context.Context, s *Service) error {
			_, err := s.TestPass(ctx, "t1", viaMCP)
			return err
		}
	}
	fail := func(viaMCP bool) func(context.Context, *Service) error {
		return func(ctx context.Context, s *Service) error {
			_, err := s.TestFail(ctx, "t1", TestReport{Actual: "broken"}, viaMCP)
			return err
		}
	}
	tests := []struct {
		name      string
		record    func(context.Context, *Service) error
		topic     string
		wantPost  string
		wantActor Actor
	}{
		{"pass over HTTP", pass(false), TopicTestPassed, "Passed by onik97", Actor{Kind: ActorKindUser, UserID: "u-1"}},
		{"pass over MCP", pass(true), TopicTestPassed, "Passed by Nexul · for onik97", Actor{Kind: ActorKindUserMCP, UserID: "u-1"}},
		{"fail over HTTP", fail(false), TopicTestFailed, "Test failed\n\n## Actual result\nbroken", Actor{Kind: ActorKindUser, UserID: "u-1"}},
		{"fail over MCP", fail(true), TopicTestFailed, "Test failed by Nexul · for onik97\n\n## Actual result\nbroken", Actor{Kind: ActorKindUserMCP, UserID: "u-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestingFixture("qa")

			err := tt.record(asUser(t.Context()), f.svc)

			require.NoError(t, err)
			require.Len(t, f.threads.posts, 1)
			assert.Equal(t, tt.wantPost, f.threads.posts[0].body)
			assert.Equal(t, "u-1", f.threads.posts[0].authorID)
			assert.Equal(t, "onik97", eventOf(t, f.repo.events, tt.topic).Payload.(TestedEvent).Tester)
			assert.Equal(t, tt.wantActor, eventOf(t, f.repo.events, TopicStatusChanged).Payload.(StatusChangedEvent).Actor)
		})
	}
}

func TestTestFail_StatusOutsideAnyColumnIsNotDone(t *testing.T) {
	f := newTestingFixture(StatusOpen)
	f.repo.tickets["t1"].Status = "legacy"

	_, err := f.svc.TestFail(asUser(t.Context()), "t1", TestReport{Actual: "broken"}, false)

	require.NoError(t, err)
}

func TestTestTarget_ResolvesFromLinkedBranches(t *testing.T) {
	f := newTestingFixture("qa")
	f.repo.branchLinks["t1"] = []BranchLink{{Owner: "acme", Repo: "app", Branch: "feature/login"}}
	f.targets.target = TestTarget{URL: "https://login.example.com", Kind: "preview", Branch: "feature/login"}

	got, err := f.svc.TestTarget(t.Context(), "t1")

	require.NoError(t, err)
	assert.Equal(t, f.targets.target, got)
	assert.Equal(t, f.repo.branchLinks["t1"], f.targets.got)
}

func TestTestTarget_Errors(t *testing.T) {
	t.Run("missing ticket", func(t *testing.T) {
		_, err := newTestingFixture("qa").svc.TestTarget(t.Context(), "nope")
		assert.ErrorIs(t, err, apperrs.ErrNotFound)
	})
	t.Run("resolution fails", func(t *testing.T) {
		f := newTestingFixture("qa")
		f.targets.err = errors.New("stacks down")
		_, err := f.svc.TestTarget(t.Context(), "t1")
		assert.ErrorContains(t, err, "stacks down")
	})
}

func TestTestingRoutes(t *testing.T) {
	f := newTestingFixture("qa")
	f.targets.target = TestTarget{URL: "https://qa.example.com", Kind: "shared", Branch: "dev"}
	routes := NewHandler(f.svc).Routes()
	serve := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(asUser(t.Context()), method, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := serve(http.MethodGet, "/api/tickets/t1/test-target", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"url":"https://qa.example.com","kind":"shared","branch":"dev"}`, rec.Body.String())

	rec = serve(http.MethodPost, "/api/tickets/t1/test/fail", `{"actual":""}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(http.MethodPost, "/api/tickets/t1/test/fail", `{bad`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(http.MethodPost, "/api/tickets/t1/test/fail", `{"actual":"blank page"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	var failed Ticket
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &failed))
	assert.Equal(t, Status("build"), failed.Status)

	f.repo.tickets["t1"].Status = "qa"
	rec = serve(http.MethodPost, "/api/tickets/t1/test/pass", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var passed Ticket
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &passed))
	assert.Equal(t, Status("shipped"), passed.Status)
	assert.Equal(t, "onik97", passed.Tester)
	assert.Equal(t, []string{"Test failed\n\n## Actual result\nblank page", "Passed by onik97 on https://qa.example.com"}, []string{f.threads.posts[0].body, f.threads.posts[1].body})

	assert.Equal(t, http.StatusNotFound, serve(http.MethodGet, "/api/tickets/nope/test-target", "").Code)
	assert.Equal(t, http.StatusNotFound, serve(http.MethodPost, "/api/tickets/nope/test/pass", "").Code)
}

func TestTestingMCPTools(t *testing.T) {
	f := newTestingFixture("qa")
	tools := map[string]func(context.Context, map[string]any) (any, error){}
	for _, tool := range MCPTools(f.svc) {
		tools[tool.Name] = tool.Call
	}
	ctx := asUser(t.Context())

	_, err := tools["ticket_get_test_target"](ctx, map[string]any{})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	got, err := tools["ticket_get_test_target"](ctx, map[string]any{"id": "t1"})
	require.NoError(t, err)
	assert.Equal(t, TestTarget{}, got)

	_, err = tools["ticket_test_fail"](ctx, map[string]any{"id": "t1", "actual": "broken", "screenshots": []any{1}})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = tools["ticket_test_fail"](ctx, map[string]any{})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	failed, err := tools["ticket_test_fail"](ctx, map[string]any{"id": "t1", "steps": "s", "expected": "e", "actual": "broken", "screenshots": []any{"att-9"}})
	require.NoError(t, err)
	assert.Equal(t, Status("build"), failed.(*Ticket).Status)
	assert.Contains(t, f.threads.posts[0].body, "![screenshot](/api/attachments/att-9)")
	assert.True(t, strings.HasPrefix(f.threads.posts[0].body, "Test failed by Nexul · for onik97\n"))

	_, err = tools["ticket_test_pass"](ctx, map[string]any{})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	passed, err := tools["ticket_test_pass"](ctx, map[string]any{"id": "t1"})
	require.NoError(t, err)
	assert.Equal(t, Status("shipped"), passed.(*Ticket).Status)
	assert.Equal(t, "Passed by Nexul · for onik97", f.threads.posts[1].body)
}
