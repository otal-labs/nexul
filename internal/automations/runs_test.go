package automations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// erroringRunsRepo makes every DeleteOlderThan call fail, to exercise
// RunCleanupLoop's error-logging branch without a real storage failure.
type erroringRunsRepo struct {
	*fakeRunsRepo
	calls atomic.Int32
}

func (r *erroringRunsRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	r.calls.Add(1)
	return 0, errors.New("db unavailable")
}

func TestAppendCappedLog_UnderCap_Appends(t *testing.T) {
	got := appendCappedLog("hello ", "world")
	assert.Equal(t, "hello world", got)
}

func TestAppendCappedLog_OverCap_Truncates(t *testing.T) {
	buf := strings.Repeat("a", maxRunLogBytes-3)
	got := appendCappedLog(buf, "xxxxxx")
	assert.Len(t, got, maxRunLogBytes)
	assert.Equal(t, buf+"xxx", got)
}

func TestAppendCappedLog_AlreadyAtCap_DiscardsChunk(t *testing.T) {
	buf := strings.Repeat("a", maxRunLogBytes)
	got := appendCappedLog(buf, "more")
	assert.Equal(t, buf, got)
}

func TestRunsService_ListByAutomation_NoPermission_ReturnsForbidden(t *testing.T) {
	svc := NewRunsService(newFakeRunsRepo(), newFakePerm(nil))
	_, err := svc.ListByAutomation(context.Background(), "u1", "a1", 10)
	require.Error(t, err)
}

func TestRunsService_ListByAutomation_EmptyActor_ReturnsUnauthorized(t *testing.T) {
	svc := NewRunsService(newFakeRunsRepo(), allowAll("u1"))
	_, err := svc.ListByAutomation(context.Background(), "", "a1", 10)
	require.Error(t, err)
}

func TestRunsService_ListByAutomation_ReturnsOnlyMatchingAutomation(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "r1", AutomationID: "a1", CreatedAt: time.Now()}))
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "r2", AutomationID: "a2", CreatedAt: time.Now()}))

	svc := NewRunsService(repo, allowAll("u1"))
	runs, err := svc.ListByAutomation(context.Background(), "u1", "a1", 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	assert.Equal(t, "r1", runs[0].ID)
}

func TestRunsService_Get_NoPermission_ReturnsForbidden(t *testing.T) {
	svc := NewRunsService(newFakeRunsRepo(), newFakePerm(nil))
	_, err := svc.Get(context.Background(), "u1", "a1", "r1")
	require.Error(t, err)
}

func TestRunsService_Get_UnknownRun_ReturnsNotFound(t *testing.T) {
	svc := NewRunsService(newFakeRunsRepo(), allowAll("u1"))
	_, err := svc.Get(context.Background(), "u1", "a1", "missing")
	require.Error(t, err)
}

func TestRunsService_Get_WrongAutomation_ReturnsNotFound(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "r1", AutomationID: "a1"}))

	svc := NewRunsService(repo, allowAll("u1"))
	_, err := svc.Get(context.Background(), "u1", "a2", "r1")
	require.Error(t, err)
}

func TestRunsService_Get_Matching_ReturnsRun(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "r1", AutomationID: "a1", Logs: "hi"}))

	svc := NewRunsService(repo, allowAll("u1"))
	run, err := svc.Get(context.Background(), "u1", "a1", "r1")
	require.NoError(t, err)
	assert.Equal(t, "hi", run.Logs)
}

func TestRunsService_Cleanup_DeletesOldRuns(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "old", AutomationID: "a1", CreatedAt: time.Now().Add(-40 * 24 * time.Hour)}))
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "new", AutomationID: "a1", CreatedAt: time.Now()}))

	svc := NewRunsService(repo, allowAll("u1"))
	n, err := svc.Cleanup(context.Background(), 30*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	_, err = repo.Get(context.Background(), "old")
	require.Error(t, err)
	_, err = repo.Get(context.Background(), "new")
	require.NoError(t, err)
}

func TestRunCleanupLoop_StopsOnContextCancel(t *testing.T) {
	svc := NewRunsService(newFakeRunsRepo(), allowAll("u1"))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunCleanupLoop(ctx, svc, time.Hour, 5*time.Millisecond, testLogger())
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup loop did not stop on context cancel")
	}
}

func TestRunCleanupLoop_DeletesOnTick(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "old", AutomationID: "a1", CreatedAt: time.Now().Add(-40 * 24 * time.Hour)}))
	svc := NewRunsService(repo, allowAll("u1"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go RunCleanupLoop(ctx, svc, 30*24*time.Hour, 5*time.Millisecond, testLogger())

	require.Eventually(t, func() bool {
		_, err := repo.Get(context.Background(), "old")
		return errors.Is(err, apperrs.ErrNotFound)
	}, 2*time.Second, 5*time.Millisecond)
}

func TestRunCleanupLoop_CleanupError_LogsAndContinues(t *testing.T) {
	repo := &erroringRunsRepo{fakeRunsRepo: newFakeRunsRepo()}
	svc := NewRunsService(repo, allowAll("u1"))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunCleanupLoop(ctx, svc, time.Hour, 5*time.Millisecond, testLogger())
		close(done)
	}()

	require.Eventually(t, func() bool { return repo.calls.Load() > 0 }, time.Second, 5*time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cleanup loop did not stop after an error tick")
	}
}

func withActor(r *http.Request, id string) *http.Request {
	return r.WithContext(identity.WithActor(r.Context(), identity.Actor{ID: id}))
}

func TestRunsHandler_List_ReturnsRuns(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "r1", AutomationID: "a1", CreatedAt: time.Now()}))
	h := NewRunsHandler(NewRunsService(repo, allowAll("u1")))

	req := withActor(httptest.NewRequest(http.MethodGet, "/api/automations/a1/runs", nil), "u1")
	req.SetPathValue("id", "a1")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var runs []Run
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &runs))
	require.Len(t, runs, 1)
}

func TestRunsHandler_List_Forbidden_ReturnsError(t *testing.T) {
	h := NewRunsHandler(NewRunsService(newFakeRunsRepo(), newFakePerm(nil)))

	req := withActor(httptest.NewRequest(http.MethodGet, "/api/automations/a1/runs", nil), "u1")
	req.SetPathValue("id", "a1")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRunsHandler_Get_UnknownRun_ReturnsNotFound(t *testing.T) {
	h := NewRunsHandler(NewRunsService(newFakeRunsRepo(), allowAll("u1")))

	req := withActor(httptest.NewRequest(http.MethodGet, "/api/automations/a1/runs/missing", nil), "u1")
	req.SetPathValue("id", "a1")
	req.SetPathValue("runID", "missing")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRunsHandler_Get_Found_ReturnsRun(t *testing.T) {
	repo := newFakeRunsRepo()
	require.NoError(t, repo.Create(context.Background(), &Run{ID: "r1", AutomationID: "a1", Logs: "log line"}))
	h := NewRunsHandler(NewRunsService(repo, allowAll("u1")))

	req := withActor(httptest.NewRequest(http.MethodGet, "/api/automations/a1/runs/r1", nil), "u1")
	req.SetPathValue("id", "a1")
	req.SetPathValue("runID", "r1")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var run Run
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &run))
	assert.Equal(t, "log line", run.Logs)
}

func TestRunsHandler_List_DefaultAndCustomLimit(t *testing.T) {
	repo := newFakeRunsRepo()
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(context.Background(), &Run{ID: string(rune('a' + i)), AutomationID: "a1", CreatedAt: time.Now()}))
	}
	h := NewRunsHandler(NewRunsService(repo, allowAll("u1")))

	req := withActor(httptest.NewRequest(http.MethodGet, "/api/automations/a1/runs?limit=2", nil), "u1")
	req.SetPathValue("id", "a1")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)

	var runs []Run
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &runs))
	assert.Len(t, runs, 2)
}
