package deploy

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeErrorRateRepo keeps episodes and the outbox rows it was handed, and refuses a second open episode per stack.
type fakeErrorRateRepo struct {
	mu     sync.Mutex
	rows   []*ErrorRateEpisode
	outbox []eventbus.OutboxEvent

	openErr, peakErr, closeErr, getErr error
}

func (f *fakeErrorRateRepo) Open(_ context.Context, e *ErrorRateEpisode, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.openErr != nil {
		return f.openErr
	}
	for _, r := range f.rows {
		if r.StackID == e.StackID && r.EndedAt == nil {
			return apperrs.ErrConflict
		}
	}
	cp := *e
	f.rows = append(f.rows, &cp)
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeErrorRateRepo) UpdatePeak(_ context.Context, e *ErrorRateEpisode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.peakErr != nil {
		return f.peakErr
	}
	for _, r := range f.rows {
		if r.ID == e.ID {
			r.PeakPercent, r.Requests, r.Errors = e.PeakPercent, e.Requests, e.Errors
		}
	}
	return nil
}

func (f *fakeErrorRateRepo) Close(_ context.Context, id string, endedAt time.Time, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closeErr != nil {
		return f.closeErr
	}
	for _, r := range f.rows {
		if r.ID == id && r.EndedAt == nil {
			r.EndedAt = &endedAt
			f.outbox = append(f.outbox, evts...)
			return nil
		}
	}
	return apperrs.ErrNotFound
}

func (f *fakeErrorRateRepo) GetOpen(_ context.Context, stackID string) (*ErrorRateEpisode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, r := range f.rows {
		if r.StackID == stackID && r.EndedAt == nil {
			cp := *r
			return &cp, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeErrorRateRepo) LatestEnd(_ context.Context, stackID string) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var last time.Time
	for _, r := range f.rows {
		if r.StackID == stackID && r.EndedAt != nil && r.EndedAt.After(last) {
			last = *r.EndedAt
		}
	}
	return last, nil
}

func (f *fakeErrorRateRepo) events(topic string) []ErrorRateEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []ErrorRateEvent
	for _, e := range f.outbox {
		if e.Topic == topic {
			out = append(out, e.Payload.(ErrorRateEvent))
		}
	}
	return out
}

type errorRateEnv struct {
	*requestEnv
	episodes *fakeErrorRateRepo
}

// newErrorRateEnv is the request counting stack with its rule on (5% over 5 minutes, 20 requests), a hostname behind a
// proxy, and episode storage. The clock reads 10:42:30, so the rule reads the minutes 10:35 to 10:39.
func newErrorRateEnv(t *testing.T) *errorRateEnv {
	t.Helper()
	env := &errorRateEnv{requestEnv: newRequestEnv(t, proxyRoute), episodes: &fakeErrorRateRepo{}}
	env.svc.SetErrorRateRepo(env.episodes)
	env.svc.projects.(*fakeProjects).exists["proj-1"] = true
	return env
}

// minutes stores one row per window minute, oldest first: requests and errors, or nothing for a minute nobody counted.
func (e *errorRateEnv) minutes(t *testing.T, window ...*[2]int64) {
	t.Helper()
	end := e.now.Truncate(time.Minute).Add(-requestLag)
	start := end.Add(-time.Duration(len(window)) * time.Minute)
	for i, w := range window {
		m := start.Add(time.Duration(i) * time.Minute)
		if w == nil {
			require.NoError(t, e.repo.MarkMissing(t.Context(), []string{"svc-1"}, m))
			continue
		}
		require.NoError(t, e.repo.Add(t.Context(), []RequestAdd{{StackID: "svc-1", Minute: m, Requests: w[0], Errors: w[1]}}))
	}
}

// each is n minutes of the same counts.
func each(n int, requests, errs int64) []*[2]int64 {
	out := make([]*[2]int64, n)
	for i := range out {
		out[i] = &[2]int64{requests, errs}
	}
	return out
}

func (e *errorRateEnv) evaluate(t *testing.T) {
	t.Helper()
	e.svc.evaluateErrorRates(t.Context(), slog.New(slog.DiscardHandler))
}

func (e *errorRateEnv) open(t *testing.T) *ErrorRateEpisode {
	t.Helper()
	open, err := e.episodes.GetOpen(t.Context(), "svc-1")
	require.NoError(t, err)
	return open
}

func TestEvaluateErrorRates_Crossing(t *testing.T) {
	tests := []struct {
		name   string
		window []*[2]int64
		opens  bool
	}{
		{"a share over the limit with enough requests crosses", each(5, 100, 10), true},
		{"a share exactly at the limit does not cross", each(5, 100, 5), false},
		{"a share just over the limit crosses", each(5, 100, 6), true},
		{"fewer requests than the floor never cross, however bad", each(5, 3, 3), false},
		{"exactly the floor crosses", each(5, 4, 4), true},
		{"a quiet window does not cross", each(5, 0, 0), false},
		{"one missing minute blocks a crossing", append(each(4, 100, 50), nil), false},
		{"errors in the first minute only still count over the window", append([]*[2]int64{{100, 100}}, each(4, 100, 0)...), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newErrorRateEnv(t)
			env.minutes(t, tt.window...)

			env.evaluate(t)

			_, err := env.episodes.GetOpen(t.Context(), "svc-1")
			assert.Equal(t, tt.opens, err == nil)
			assert.Len(t, env.episodes.events(TopicErrorRateCrossed), map[bool]int{true: 1, false: 0}[tt.opens])
		})
	}
}

func TestEvaluateErrorRates_TheCrossingEventCarriesTheWindowAndTheRule(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 12)...)

	env.evaluate(t)

	got := env.episodes.events(TopicErrorRateCrossed)
	require.Len(t, got, 1)
	assert.Equal(t, ErrorRateEvent{
		WorkspaceID: "ws-1", ProjectID: "proj-1", StackID: "svc-1", StackName: "api", EpisodeID: got[0].EpisodeID,
		PeakPercent: 12, Requests: 500, Errors: 60, LimitPercent: 5, WindowMinutes: 5, MinRequests: 20, StartedAt: env.now,
	}, got[0])
	assert.NotEmpty(t, got[0].EpisodeID)
}

func TestEvaluateErrorRates_AnOpenEpisodeIsNotOpenedAgain(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 10)...)

	env.evaluate(t)
	env.evaluate(t)

	assert.Len(t, env.episodes.events(TopicErrorRateCrossed), 1)
}

func TestEvaluateErrorRates_Recovery(t *testing.T) {
	tests := []struct {
		name      string
		window    []*[2]int64
		recovered bool
	}{
		{"five minutes under the limit recover", each(5, 100, 1), true},
		{"quiet minutes are under the limit", each(5, 0, 0), true},
		{"a minute exactly at the limit is under it", each(5, 100, 5), true},
		{"one minute over the limit keeps it open", append(each(4, 100, 1), &[2]int64{100, 6}), false},
		{"an old bad minute in the window keeps it open", append([]*[2]int64{{100, 40}}, each(4, 100, 1)...), false},
		{"a missing minute neither recovers it", append(each(4, 100, 0), nil), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newErrorRateEnv(t)
			env.minutes(t, each(5, 100, 20)...)
			env.evaluate(t)
			require.Len(t, env.episodes.events(TopicErrorRateCrossed), 1)

			env.now = env.now.Add(5 * time.Minute)
			env.minutes(t, tt.window...)
			env.evaluate(t)

			recovered := env.episodes.events(TopicErrorRateRecovered)
			assert.Equal(t, tt.recovered, len(recovered) == 1)
			if !tt.recovered {
				return
			}
			require.NotNil(t, recovered[0].EndedAt)
			assert.Equal(t, env.now, *recovered[0].EndedAt)
			assert.EqualValues(t, 20, recovered[0].PeakPercent)
			_, err := env.episodes.GetOpen(t.Context(), "svc-1")
			assert.ErrorIs(t, err, apperrs.ErrNotFound)
		})
	}
}

func TestEvaluateErrorRates_AfterARecoveryTheNextCrossingIsANewEpisode(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 20)...)
	env.evaluate(t)
	env.now = env.now.Add(5 * time.Minute)
	env.minutes(t, each(5, 100, 0)...)
	env.evaluate(t)
	env.now = env.now.Add(5 * time.Minute)
	env.minutes(t, each(5, 100, 30)...)
	env.evaluate(t)

	crossed := env.episodes.events(TopicErrorRateCrossed)
	require.Len(t, crossed, 2)
	assert.NotEqual(t, crossed[0].EpisodeID, crossed[1].EpisodeID)
}

func TestEvaluateErrorRates_PeakOnlyRisesAndOnlyOnACompleteWindow(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 10)...)
	env.evaluate(t)
	assert.EqualValues(t, 10, env.open(t).PeakPercent)

	step := func(window ...*[2]int64) {
		env.now = env.now.Add(5 * time.Minute)
		env.minutes(t, window...)
		env.evaluate(t)
	}
	step(each(5, 100, 25)...)
	open := env.open(t)
	assert.EqualValues(t, 25, open.PeakPercent)
	assert.EqualValues(t, 500, open.Requests)
	assert.EqualValues(t, 125, open.Errors)

	step(each(5, 100, 8)...)
	assert.EqualValues(t, 25, env.open(t).PeakPercent, "a better window never lowers the peak")

	step(append(each(4, 100, 90), nil)...)
	assert.EqualValues(t, 25, env.open(t).PeakPercent, "a window with a gap is not read")
}

func TestEvaluateErrorRates_OnlyStacksWithTheRuleOnAreRead(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 50)...)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	stack.ErrorRate.Enabled = false
	require.NoError(t, env.svc.stacks.Update(t.Context(), stack))

	env.evaluate(t)

	assert.Empty(t, env.episodes.events(TopicErrorRateCrossed))
}

func TestEvaluateErrorRates_TheRulesOwnNumbersDecide(t *testing.T) {
	env := newErrorRateEnv(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	stack.ErrorRate = ErrorRateRule{Enabled: true, Percent: 50, Minutes: 2, MinRequests: 1}
	require.NoError(t, env.svc.stacks.Update(t.Context(), stack))
	env.minutes(t, &[2]int64{10, 9}, &[2]int64{10, 2})

	env.evaluate(t)

	got := env.episodes.events(TopicErrorRateCrossed)
	require.Len(t, got, 1)
	assert.EqualValues(t, 55, got[0].PeakPercent)
	assert.Equal(t, 2, got[0].WindowMinutes)
}

func TestEvaluateErrorRates_FailuresAreLoggedAndTheNextStackIsStillRead(t *testing.T) {
	env := newErrorRateEnv(t)
	other := validStack()
	other.ID, other.Slug, other.Name = "svc-0", "web", "web"
	other.ErrorRate = ErrorRateRule{Enabled: true, Percent: 5, Minutes: 5, MinRequests: 20}
	require.NoError(t, env.svc.stacks.Create(t.Context(), &other))
	env.minutes(t, each(5, 100, 10)...)
	var logged bytes.Buffer

	env.episodes.getErr = errors.New("db down")
	env.svc.evaluateErrorRates(t.Context(), slog.New(slog.NewTextHandler(&logged, nil)))
	assert.Contains(t, logged.String(), "error-rate evaluation failed")
	assert.Contains(t, logged.String(), "svc-0", "the first stack failed")
	assert.Contains(t, logged.String(), "svc-1", "and the second was still tried")

	env.episodes.getErr = nil
	env.episodes.openErr = errors.New("db down")
	logged.Reset()
	env.svc.evaluateErrorRates(t.Context(), slog.New(slog.NewTextHandler(&logged, nil)))
	assert.Contains(t, logged.String(), "open error-rate episode")
	assert.Empty(t, env.episodes.rows)
}

func TestEvaluateErrorRates_WritesFailuresOnAnOpenEpisodeAreReturned(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 10)...)
	env.evaluate(t)
	end := env.now.Truncate(time.Minute).Add(-requestLag)

	env.episodes.peakErr = errors.New("db down")
	env.now = env.now.Add(5 * time.Minute)
	env.minutes(t, each(5, 100, 40)...)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	assert.ErrorContains(t, env.svc.evaluateErrorRate(t.Context(), stack, env.now.Truncate(time.Minute).Add(-requestLag)), "update error-rate episode")

	env.episodes.closeErr = errors.New("db down")
	env.minutes(t, each(5, 100, 0)...)
	env.now = env.now.Add(5 * time.Minute)
	env.minutes(t, each(5, 100, 0)...)
	assert.ErrorContains(t, env.svc.evaluateErrorRate(t.Context(), stack, env.now.Truncate(time.Minute).Add(-requestLag)), "close error-rate episode")
	assert.True(t, end.Before(env.now))
}

func TestEvaluateErrorRates_UnwiredOrUnreadableDoesNothing(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 10)...)
	env.svc.errorRates = nil
	env.svc.evaluateErrorRates(t.Context(), slog.New(slog.DiscardHandler))
	assert.Empty(t, env.episodes.rows)

	env.svc.SetErrorRateRepo(env.episodes)
	env.svc.stacks = failingList{env.svc.stacks}
	var logged bytes.Buffer
	env.svc.evaluateErrorRates(t.Context(), slog.New(slog.NewTextHandler(&logged, nil)))
	assert.Contains(t, logged.String(), "list stacks with an error-rate rule failed")
}

type failingList struct{ StackRepo }

func (failingList) ListWithErrorRate(context.Context) ([]*Stack, error) {
	return nil, errors.New("db down")
}

func TestRunRequestCounts_EvaluatesTheRulesAfterEachPoll(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 10)...)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		env.svc.RunRequestCounts(ctx, slog.New(slog.DiscardHandler))
		close(done)
	}()
	cancel()
	<-done

	assert.Len(t, env.episodes.events(TopicErrorRateCrossed), 1, "the first pass runs before it waits")
}

func TestErrorRateState(t *testing.T) {
	env := newErrorRateEnv(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)

	open, recovered, err := env.svc.ErrorRateState(t.Context(), stack)
	require.NoError(t, err)
	assert.Nil(t, open)
	assert.Nil(t, recovered, "a stack that never crossed has neither")

	env.minutes(t, each(5, 100, 10)...)
	env.evaluate(t)
	open, recovered, err = env.svc.ErrorRateState(t.Context(), stack)
	require.NoError(t, err)
	require.NotNil(t, open)
	assert.Nil(t, recovered)

	env.now = env.now.Add(5 * time.Minute)
	env.minutes(t, each(5, 100, 0)...)
	env.evaluate(t)
	open, recovered, err = env.svc.ErrorRateState(t.Context(), stack)
	require.NoError(t, err)
	assert.Nil(t, open)
	require.NotNil(t, recovered, "it ended just now")
	assert.Equal(t, env.now, *recovered)

	env.now = env.now.Add(2 * time.Hour)
	_, recovered, err = env.svc.ErrorRateState(t.Context(), stack)
	require.NoError(t, err)
	assert.Nil(t, recovered, "an hour later it is old news")

	env.now = env.now.Add(-2 * time.Hour)
	off := *stack
	off.ErrorRate.Enabled = false
	open, recovered, err = env.svc.ErrorRateState(t.Context(), &off)
	require.NoError(t, err)
	assert.Nil(t, open)
	assert.Nil(t, recovered, "with the rule off there is nothing to show")

	env.episodes.getErr = errors.New("db down")
	_, _, err = env.svc.ErrorRateState(t.Context(), stack)
	assert.Error(t, err)
}

func TestWantsCounts_OnlyAStackWithTheRuleOnIsCounted(t *testing.T) {
	env := newRequestEnv(t, proxyRoute, tunnelRoute)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	stack.ErrorRate.Enabled = false
	require.NoError(t, env.svc.stacks.Update(t.Context(), stack))

	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@docker", Requests: 9, Errors: 9})
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.New(slog.DiscardHandler)))

	assert.Nil(t, env.repo.at("svc-1", clock(10, 42)), "the proxy frame adds nothing for a stack with the rule off")
	assert.Empty(t, env.sources.calls, "and Cloudflare is not asked about it")
}

func TestRequestWindow_ExplainsWhyTheRuleCannotWatchAStack(t *testing.T) {
	env := newErrorRateEnv(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)

	w, err := env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.Empty(t, w.ErrorRateUnavailable, "a proxy hostname is counted by the machine's own proxy")

	env.sources.routes = nil
	w, err = env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.Equal(t, "Needs a hostname on this stack", w.ErrorRateUnavailable)
	assert.False(t, w.Counted)

	env.sources.routes = []RequestRoute{tunnelRoute}
	w, err = env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.Empty(t, w.ErrorRateUnavailable, "a tunnel hostname whose analytics answer is readable")
	require.Len(t, env.sources.calls, 1)
	assert.Equal(t, "z1", env.sources.calls[0].zone)
}

func TestRequestWindow_ATokenThatCannotReadAnalyticsIsAskedOnceAMinute(t *testing.T) {
	env := newErrorRateEnv(t)
	env.sources.routes = []RequestRoute{tunnelRoute}
	env.sources.zoneErr = map[string]error{"z1": apperrs.ErrInvalid}
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)

	for range 3 {
		w, err := env.svc.RequestWindow(t.Context(), stack)
		require.NoError(t, err)
		assert.Equal(t, "Your Cloudflare token can't read analytics; add Zone Analytics Read to it", w.ErrorRateUnavailable)
		assert.True(t, w.Counted, "the hostname is still shown")
	}
	assert.Len(t, env.sources.calls, 1, "the answer is reused within the minute")

	env.now = env.now.Add(analyticsProbeTTL)
	env.sources.zoneErr = nil
	w, err := env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.Empty(t, w.ErrorRateUnavailable, "asked again after the minute, and the token works now")
	assert.Len(t, env.sources.calls, 2)

	env.now = env.now.Add(analyticsProbeTTL)
	env.sources.zoneErr = map[string]error{"z1": errors.New("cloudflare down")}
	w, err = env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.Empty(t, w.ErrorRateUnavailable, "an outage at Cloudflare is not a refusal")
}

func TestUpdateStack_TheRuleNeedsAStackItCanWatch(t *testing.T) {
	env := newErrorRateEnv(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	off := *stack
	off.ErrorRate.Enabled = false
	require.NoError(t, env.svc.stacks.Update(t.Context(), &off))
	on := off
	on.ErrorRate.Enabled = true

	env.sources.routes = nil
	_, err = env.svc.UpdateStack(t.Context(), on)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "Needs a hostname on this stack")

	env.sources.routes = []RequestRoute{tunnelRoute}
	env.sources.zoneErr = map[string]error{"z1": apperrs.ErrInvalid}
	_, err = env.svc.UpdateStack(t.Context(), on)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.ErrorContains(t, err, "can't read analytics")

	env.sources.routes = []RequestRoute{proxyRoute}
	updated, err := env.svc.UpdateStack(t.Context(), on)
	require.NoError(t, err)
	assert.True(t, updated.ErrorRate.Enabled)

	env.sources.routes = nil
	_, err = env.svc.UpdateStack(t.Context(), *updated)
	assert.NoError(t, err, "a rule that is already on is not refused when the hostname goes away; it can still be changed or switched off")
}

func TestUpdateStack_TheRulesNumbersAreHeldToSaneBounds(t *testing.T) {
	tests := []struct {
		name string
		rule ErrorRateRule
		ok   bool
	}{
		{"the defaults", ErrorRateRule{Percent: 5, Minutes: 5, MinRequests: 20}, true},
		{"zero numbers mean the defaults", ErrorRateRule{Enabled: true}, true},
		{"the widest rule", ErrorRateRule{Percent: 100, Minutes: 60, MinRequests: 1_000_000}, true},
		{"the tightest rule", ErrorRateRule{Percent: 0.1, Minutes: 1, MinRequests: 1}, true},
		{"a negative limit", ErrorRateRule{Percent: -1, Minutes: 5, MinRequests: 20}, false},
		{"a limit under a tenth of a percent", ErrorRateRule{Percent: 0.01, Minutes: 5, MinRequests: 20}, false},
		{"a limit over a hundred percent", ErrorRateRule{Percent: 101, Minutes: 5, MinRequests: 20}, false},
		{"a window longer than the hour of counts kept on the page", ErrorRateRule{Percent: 5, Minutes: 61, MinRequests: 20}, false},
		{"a negative window", ErrorRateRule{Percent: 5, Minutes: -2, MinRequests: 20}, false},
		{"a negative floor", ErrorRateRule{Percent: 5, Minutes: 5, MinRequests: -1}, false},
		{"a floor no stack could meet", ErrorRateRule{Percent: 5, Minutes: 5, MinRequests: 1_000_001}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := validStack()
			stack.ErrorRate = tt.rule
			err := stack.Validate()
			if tt.ok {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}

func TestUpdateStack_AMissingRuleKeepsTheDefaultsAndOff(t *testing.T) {
	env := newErrorRateEnv(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	stack.ErrorRate = ErrorRateRule{}

	updated, err := env.svc.UpdateStack(t.Context(), *stack)

	require.NoError(t, err)
	assert.Equal(t, ErrorRateRule{Percent: 5, Minutes: 5, MinRequests: 20}, updated.ErrorRate)
}

func TestUpdateStack_SwitchingTheRuleOffEndsAnOpenEpisodeWithoutARecoveredEvent(t *testing.T) {
	env := newErrorRateEnv(t)
	env.minutes(t, each(5, 100, 10)...)
	env.evaluate(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	off := *stack
	off.ErrorRate.Enabled = false

	_, err = env.svc.UpdateStack(t.Context(), off)

	require.NoError(t, err)
	_, err = env.episodes.GetOpen(t.Context(), "svc-1")
	assert.ErrorIs(t, err, apperrs.ErrNotFound, "nothing watches it any more")
	assert.Empty(t, env.episodes.events(TopicErrorRateRecovered), "the share did not recover, so nobody is told it did")

	again := off
	_, err = env.svc.UpdateStack(t.Context(), again)
	assert.NoError(t, err, "switching off a rule with no episode is fine")

	env.episodes.getErr = errors.New("db down")
	on, off2 := off, off
	on.ErrorRate.Enabled = true
	_, err = env.svc.UpdateStack(t.Context(), on)
	require.NoError(t, err)
	_, err = env.svc.UpdateStack(t.Context(), off2)
	assert.ErrorContains(t, err, "get open error-rate episode")
}

func TestStackGet_ReturnsTheRuleTheOpenEpisodeAndWhyItCannotWatch(t *testing.T) {
	s, stack, _ := stackFixture(t)
	repo := newFakeRequestRepo()
	sources := &fakeRequestSources{}
	s.SetRequestCounts(sources, repo)
	episodes := &fakeErrorRateRepo{}
	s.SetErrorRateRepo(episodes)

	got, err := call(t, s, "stack_get", `{"id":"`+stack.ID+`"}`)
	require.NoError(t, err)
	detail := got.(stackDetail)
	assert.Equal(t, ErrorRateRule{Percent: 5, Minutes: 5, MinRequests: 20}, detail.ErrorRate)
	assert.Equal(t, "Needs a hostname on this stack", detail.ErrorRateUnavailable)
	assert.Nil(t, detail.OpenErrorRateEpisode)

	update, err := call(t, s, "stack_update", `{"id":"`+stack.ID+`","error_rate":{"enabled":true,"percent":2.5,"min_requests":50}}`)
	assert.Nil(t, update)
	assert.ErrorContains(t, err, "Needs a hostname on this stack")

	containers, err := s.ListServices(t.Context(), stack.ID)
	require.NoError(t, err)
	sources.routes = []RequestRoute{{Router: "nexul-e1", ContainerID: containers[0].ID, Machine: stack.Machine}}
	update, err = call(t, s, "stack_update", `{"id":"`+stack.ID+`","error_rate":{"enabled":true,"percent":2.5,"min_requests":50}}`)
	require.NoError(t, err)
	assert.Equal(t, ErrorRateRule{Enabled: true, Percent: 2.5, Minutes: 5, MinRequests: 50}, update.(stackUpdated).ErrorRate, "the numbers not sent keep their values")

	update, err = call(t, s, "stack_update", `{"id":"`+stack.ID+`","crash_tickets":true}`)
	require.NoError(t, err)
	assert.Equal(t, ErrorRateRule{Enabled: true, Percent: 2.5, Minutes: 5, MinRequests: 50}, update.(stackUpdated).ErrorRate, "an update that does not name the rule leaves it")

	_, err = call(t, s, "stack_update", `{"id":"`+stack.ID+`","error_rate":{"minutes":90}}`)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)

	started := time.Date(2026, 8, 2, 11, 55, 0, 0, time.UTC)
	require.NoError(t, episodes.Open(t.Context(), &ErrorRateEpisode{
		ID: "ep-1", StackID: stack.ID, StartedAt: started, PeakPercent: 7.2, Requests: 400, Errors: 29, LimitPercent: 2.5, WindowMinutes: 5, MinRequests: 50,
	}))
	got, err = call(t, s, "stack_get", `{"id":"`+stack.ID+`"}`)
	require.NoError(t, err)
	open := got.(stackDetail).OpenErrorRateEpisode
	require.NotNil(t, open)
	assert.Equal(t, &errorRateResult{Since: started, PeakPercent: 7.2, Requests: 400, Errors5xx: 29, LimitPercent: 2.5, WindowMinutes: 5}, open)
	assert.Contains(t, asJSON(t, got), `"open_error_rate_episode":{"since":"2026-08-02T11:55:00Z","peak_percent":7.2`)

	episodes.getErr = errors.New("db down")
	_, err = call(t, s, "stack_get", `{"id":"`+stack.ID+`"}`)
	assert.Error(t, err)
}

func TestGetStackRoute_CarriesTheOpenEpisodeAndTheRecovery(t *testing.T) {
	env := newErrorRateEnv(t)
	h := NewHandler(env.svc).Routes()
	get := func() stackResponse {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodGet, "/api/stacks/svc-1", nil)))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var resp stackResponse
		decodeDeploy(t, rec, &resp)
		return resp
	}

	resp := get()
	assert.Nil(t, resp.ErrorRateEpisode)
	assert.Nil(t, resp.ErrorRateRecoveredAt)
	assert.True(t, resp.ErrorRate.Enabled)

	env.minutes(t, each(5, 100, 10)...)
	env.evaluate(t)
	resp = get()
	require.NotNil(t, resp.ErrorRateEpisode)
	assert.EqualValues(t, 10, resp.ErrorRateEpisode.PeakPercent)

	env.now = env.now.Add(5 * time.Minute)
	env.minutes(t, each(5, 100, 0)...)
	env.evaluate(t)
	resp = get()
	assert.Nil(t, resp.ErrorRateEpisode)
	require.NotNil(t, resp.ErrorRateRecoveredAt)
	assert.Equal(t, env.now, *resp.ErrorRateRecoveredAt)

	env.episodes.getErr = errors.New("db down")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withActor(httptest.NewRequest(http.MethodGet, "/api/stacks/svc-1", nil)))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func errorRateEventFor(t *testing.T, topic string, mutate ...func(*ErrorRateEvent)) eventbus.Event {
	t.Helper()
	back := time.Date(2026, 9, 1, 12, 20, 0, 0, time.UTC)
	p := ErrorRateEvent{
		ProjectID: "proj-1", StackID: "svc-1", StackName: "api", EpisodeID: "ep-1", PeakPercent: 12.34, Requests: 500, Errors: 62,
		LimitPercent: 5, WindowMinutes: 5, MinRequests: 20, StartedAt: time.Date(2026, 9, 1, 12, 2, 0, 0, time.UTC),
	}
	if topic == TopicErrorRateRecovered {
		p.EndedAt = &back
	}
	for _, m := range mutate {
		m(&p)
	}
	return busEvent(t, topic, p)
}

func TestErrorRateCrashTickets_OneTicketPerEpisodeWhileNoneIsOpen(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed)))

	require.Len(t, f.desk.filed, 1)
	got := f.desk.filed[0]
	assert.Equal(t, "proj-1", got.project)
	assert.Equal(t, "api errors: 5xx at 12.3%", got.title)
	for _, want := range []string{"`api`", "12.3%", "62 of 500", "5 minutes", "limit of 5%", "2026-09-01T12:02:00Z", "ep-1", "`stack_get`"} {
		assert.Contains(t, got.body, want)
	}
	assert.Equal(t, "t1", f.links.links["svc-1/"+errorRateSubject])

	require.NoError(t, f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed, func(p *ErrorRateEvent) { p.EpisodeID = "ep-2" })))
	assert.Len(t, f.desk.filed, 1, "a repeat while the ticket is open")
	require.Len(t, f.desk.comments["t1"], 1)
	assert.Contains(t, f.desk.comments["t1"][0], "ep-2")

	f.desk.closed["t1"] = true
	require.NoError(t, f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed, func(p *ErrorRateEvent) { p.EpisodeID = "ep-3" })))
	assert.Len(t, f.desk.filed, 2, "once the ticket is done the next episode opens a new one")
	assert.Equal(t, "t2", f.links.links["svc-1/"+errorRateSubject])
}

func TestErrorRateCrashTickets_AServiceCrashAndAnEpisodeKeepSeparateTickets(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleContainerStopped(t.Context(), outageEvent(t, TopicContainerStopped)))
	require.NoError(t, f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed)))

	assert.Len(t, f.desk.filed, 2)
	assert.Empty(t, f.desk.comments)
}

func TestErrorRateCrashTickets_RecoveryCommentsAndLeavesTheTicketOpen(t *testing.T) {
	f := newCrashFixture(t)
	require.NoError(t, f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed)))
	require.NoError(t, f.s.HandleErrorRateRecovered(t.Context(), errorRateEventFor(t, TopicErrorRateRecovered)))

	require.Len(t, f.desk.comments["t1"], 1)
	for _, want := range []string{"back under 5% at 2026-09-01T12:20:00Z", "2026-09-01T12:02:00Z", "12.3%", "stays open"} {
		assert.Contains(t, f.desk.comments["t1"][0], want)
	}
	assert.False(t, f.desk.closed["t1"])
}

func TestErrorRateCrashTickets_DoNothingWhere(t *testing.T) {
	tests := []struct {
		name   string
		stack  func(*Stack)
		detach bool
		event  func(*ErrorRateEvent)
	}{
		{"the switch is off", func(s *Stack) { s.CrashTickets = false }, false, nil},
		{"the stack has no project", func(s *Stack) { s.ProjectID = "" }, false, nil},
		{"the event names a stack that is gone", nil, false, func(p *ErrorRateEvent) { p.StackID = "gone" }},
		{"no ticket desk is wired", nil, true, nil},
	}
	for _, tt := range tests {
		for _, topic := range []string{TopicErrorRateCrossed, TopicErrorRateRecovered} {
			t.Run(tt.name+" "+topic, func(t *testing.T) {
				var mutate []func(*Stack)
				if tt.stack != nil {
					mutate = append(mutate, tt.stack)
				}
				f := newCrashFixture(t, mutate...)
				f.links.links["svc-1/"+errorRateSubject] = "t9"
				if tt.detach {
					f.s.SetCrashTickets(nil, nil)
				}
				var mutateEvent []func(*ErrorRateEvent)
				if tt.event != nil {
					mutateEvent = append(mutateEvent, tt.event)
				}
				handle := f.s.HandleErrorRateCrossed
				if topic == TopicErrorRateRecovered {
					handle = f.s.HandleErrorRateRecovered
				}
				require.NoError(t, handle(t.Context(), errorRateEventFor(t, topic, mutateEvent...)))
				assert.Empty(t, f.desk.filed)
				assert.Empty(t, f.desk.comments)
			})
		}
	}
}

func TestErrorRateCrashTickets_ErrorPaths(t *testing.T) {
	boom := errors.New("boom")
	f := newCrashFixture(t)
	f.desk.fileErr = boom
	require.ErrorIs(t, f.s.HandleErrorRateCrossed(t.Context(), errorRateEventFor(t, TopicErrorRateCrossed)), boom)

	f = newCrashFixture(t)
	f.links.links["svc-1/"+errorRateSubject] = "t9"
	f.desk.postErr = boom
	require.ErrorIs(t, f.s.HandleErrorRateRecovered(t.Context(), errorRateEventFor(t, TopicErrorRateRecovered)), boom)

	for _, handle := range []func(context.Context, eventbus.Event) error{f.s.HandleErrorRateCrossed, f.s.HandleErrorRateRecovered} {
		err := handle(t.Context(), eventbus.Event{Topic: TopicErrorRateCrossed, Payload: []byte("{")})
		require.ErrorIs(t, err, apperrs.ErrFatal)
	}
}

func TestTopics_PublishTheErrorRateEvents(t *testing.T) {
	var got []string
	for _, topic := range Topics() {
		if _, ok := topic.Payload.(ErrorRateEvent); ok {
			got = append(got, topic.Name)
		}
	}
	assert.Equal(t, []string{"stack.error_rate_crossed", "stack.error_rate_recovered"}, got)
}
