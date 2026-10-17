package deploy

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRequestRepo keeps minutes the way the table does, one row per stack, source and minute: Add sums into the proxy's
// row and fills a missing one, Set replaces the tunnel's, MarkMissing leaves a row that is already there alone, and
// reads sum the sources.
type fakeRequestRepo struct {
	mu   sync.Mutex
	rows map[string]map[RequestSource]map[time.Time]*RequestMinute
	// pruned is the last cut-off Prune was given.
	pruned time.Time
}

func newFakeRequestRepo() *fakeRequestRepo {
	return &fakeRequestRepo{rows: map[string]map[RequestSource]map[time.Time]*RequestMinute{}}
}

func (f *fakeRequestRepo) row(stackID string, source RequestSource, m time.Time) *RequestMinute {
	if f.rows[stackID] == nil {
		f.rows[stackID] = map[RequestSource]map[time.Time]*RequestMinute{}
	}
	if f.rows[stackID][source] == nil {
		f.rows[stackID][source] = map[time.Time]*RequestMinute{}
	}
	return f.rows[stackID][source][m]
}

func (f *fakeRequestRepo) Add(_ context.Context, adds []RequestAdd) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range adds {
		r := f.row(a.StackID, SourceProxy, a.Minute)
		if r == nil {
			r = &RequestMinute{Minute: a.Minute}
			f.rows[a.StackID][SourceProxy][a.Minute] = r
		}
		r.Requests, r.Errors = addTo(r.Requests, a.Requests), addTo(r.Errors, a.Errors)
	}
	return nil
}

func (f *fakeRequestRepo) Set(_ context.Context, sets []RequestAdd) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range sets {
		f.row(a.StackID, SourceTunnel, a.Minute)
		requests, errs := a.Requests, a.Errors
		f.rows[a.StackID][SourceTunnel][a.Minute] = &RequestMinute{Minute: a.Minute, Requests: &requests, Errors: &errs}
	}
	return nil
}

func addTo(cur *int64, n int64) *int64 {
	if cur != nil {
		n += *cur
	}
	return &n
}

func (f *fakeRequestRepo) MarkMissing(_ context.Context, source RequestSource, stackIDs []string, minute time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range stackIDs {
		if f.row(id, source, minute) == nil {
			f.rows[id][source][minute] = &RequestMinute{Minute: minute}
		}
	}
	return nil
}

func (f *fakeRequestRepo) Since(_ context.Context, stackID string, from time.Time) ([]RequestMinute, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	merged := map[time.Time]*RequestMinute{}
	for _, minutes := range f.rows[stackID] {
		for m, r := range minutes {
			if m.Before(from) {
				continue
			}
			sum := merged[m]
			if sum == nil {
				sum = &RequestMinute{Minute: m}
				merged[m] = sum
			}
			if r.Requests != nil {
				sum.Requests, sum.Errors = addTo(sum.Requests, *r.Requests), addTo(sum.Errors, *r.Errors)
			}
		}
	}
	out := make([]RequestMinute, 0, len(merged))
	for _, r := range merged {
		out = append(out, *r)
	}
	slices.SortFunc(out, func(a, b RequestMinute) int { return a.Minute.Compare(b.Minute) })
	return out, nil
}

func (f *fakeRequestRepo) Prune(_ context.Context, before time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pruned = before
	return nil
}

// at returns the counts stored for a minute: nil when there is no row, and a row with nil counts when it is missing.
func (f *fakeRequestRepo) at(stackID string, m time.Time) *RequestMinute {
	rows, _ := f.Since(context.Background(), stackID, m)
	if len(rows) == 0 || !rows[0].Minute.Equal(m) {
		return nil
	}
	return &rows[0]
}

type tunnelCall struct {
	zone      string
	hostnames []string
	from, to  time.Time
}

type fakeRequestSources struct {
	routes    []RequestRoute
	routesErr error
	counts    map[string][]HostMinute
	zoneErr   map[string]error
	calls     []tunnelCall
}

func (f *fakeRequestSources) Routes(context.Context) ([]RequestRoute, error) {
	return f.routes, f.routesErr
}

func (f *fakeRequestSources) TunnelCounts(_ context.Context, zone string, hostnames []string, from, to time.Time) ([]HostMinute, error) {
	f.calls = append(f.calls, tunnelCall{zone, hostnames, from, to})
	if err := f.zoneErr[zone]; err != nil {
		return nil, err
	}
	return f.counts[zone], nil
}

type requestEnv struct {
	svc     *Service
	repo    *fakeRequestRepo
	sources *fakeRequestSources
	now     time.Time
}

// newRequestEnv is the shared stack svc-1 on outageMachine with the error-rate rule on, one container, c-web, and request counting wired.
func newRequestEnv(t *testing.T, routes ...RequestRoute) *requestEnv {
	t.Helper()
	stacks, containers := newFakeStackRepo(), newFakeContainerRepo()
	stack := validStack()
	stack.ErrorRate = ErrorRateRule{Enabled: true, Percent: 5, Minutes: 5, MinRequests: 20}
	require.NoError(t, stacks.Create(t.Context(), &stack))
	require.NoError(t, containers.Create(t.Context(), &Container{ID: "c-web", StackID: "svc-1", Name: "web"}))
	svc := newTestServiceWith(newFakeRepo(), stacks, containers, newFakeProjects(), newFakeBus())
	env := &requestEnv{svc: svc, repo: newFakeRequestRepo(), sources: &fakeRequestSources{routes: routes}, now: time.Date(2026, 10, 16, 10, 42, 30, 0, time.UTC)}
	svc.now = func() time.Time { return env.now }
	svc.SetRequestCounts(env.sources, env.repo)
	return env
}

var (
	proxyRoute  = RequestRoute{Router: "nexul-e1", Hostname: "app.example.com", ContainerID: "c-web", Machine: outageMachine}
	tunnelRoute = RequestRoute{Router: "nexul-e2", Hostname: "App.Example.com", ContainerID: "c-web", ZoneID: "z1", Machine: outageMachine, Tunnel: true}
)

func (e *requestEnv) counts(t *testing.T, machine string, counts ...routerCount) {
	t.Helper()
	require.NoError(t, e.svc.HandleRequestCounts(t.Context(), busEvent(t, TopicRunnerRequestCounts, requestCountsEvent{Machine: machine, Counts: counts})))
}

func clock(h, m int) time.Time { return time.Date(2026, 10, 16, h, m, 0, 0, time.UTC) }

func TestHandleRequestCounts_AddsRouterCountsToTheStacksMinute(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@file", Requests: 10, Errors: 2})
	env.now = env.now.Add(10 * time.Second)
	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@file", Requests: 5, Errors: 1})

	row := env.repo.at("svc-1", clock(10, 42))
	require.NotNil(t, row)
	assert.EqualValues(t, 15, *row.Requests)
	assert.EqualValues(t, 3, *row.Errors)
}

func TestHandleRequestCounts_IgnoresWhatIsNotAProxyRouteOfThisMachine(t *testing.T) {
	tests := []struct {
		name    string
		machine string
		router  string
	}{
		{"a router no exposure labelled", outageMachine, "api@internal"},
		{"the instance route", outageMachine, "nexul@file"},
		{"an exposure on a tunnel gateway", outageMachine, "nexul-e2@file"},
		{"another machine's frame", "elsewhere", "nexul-e1@file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newRequestEnv(t, proxyRoute, tunnelRoute)
			env.counts(t, tt.machine, routerCount{Router: tt.router, Requests: 9, Errors: 9})
			row := env.repo.at("svc-1", clock(10, 42))
			if tt.machine != outageMachine {
				assert.Nil(t, row, "a machine with no proxy gateway for the stack touches nothing")
				return
			}
			require.NotNil(t, row, "the frame arrived, so the minute was counted")
			assert.Zero(t, *row.Requests)
			assert.Zero(t, *row.Errors)
		})
	}
}

func TestHandleRequestCounts_AFrameWithNoCountsStoresAZeroMinute(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	env.counts(t, outageMachine)
	row := env.repo.at("svc-1", clock(10, 42))
	require.NotNil(t, row)
	require.NotNil(t, row.Requests, "quiet is zero, not missing")
	assert.Zero(t, *row.Requests)
}

func TestHandleRequestCounts_ErrorsAndUnwiredStayQuiet(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	env.sources.routesErr = errors.New("db down")
	require.Error(t, env.svc.HandleRequestCounts(t.Context(), busEvent(t, TopicRunnerRequestCounts, requestCountsEvent{Machine: outageMachine})))

	err := env.svc.HandleRequestCounts(t.Context(), busEvent(t, TopicRunnerRequestCounts, "not an event"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "parse")

	env.svc.requests = nil
	require.NoError(t, env.svc.HandleRequestCounts(t.Context(), busEvent(t, TopicRunnerRequestCounts, requestCountsEvent{Machine: outageMachine})))
}

func TestHandleRequestCounts_SkipsARouteWhoseContainerIsGone(t *testing.T) {
	env := newRequestEnv(t, proxyRoute, RequestRoute{Router: "nexul-e9", ContainerID: "c-ghost", Machine: outageMachine})
	env.counts(t, outageMachine, routerCount{Router: "nexul-e9@file", Requests: 4})
	assert.Nil(t, env.repo.rows["c-ghost"])
	assert.NotNil(t, env.repo.at("svc-1", clock(10, 42)))
}

func TestPollRequestCounts_WritesTheTunnelMinuteAndZeroFillsQuietOnes(t *testing.T) {
	env := newRequestEnv(t, tunnelRoute)
	// now 10:42:30, so the window ends at 10:40 and a fresh process reads the five minutes from 10:35.
	env.sources.counts = map[string][]HostMinute{"z1": {{Hostname: "app.example.com", Minute: clock(10, 39), Requests: 40, Errors: 3}}}

	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	require.Len(t, env.sources.calls, 1, "one query for the zone")
	assert.Equal(t, tunnelCall{"z1", []string{"app.example.com"}, clock(10, 35), clock(10, 40)}, env.sources.calls[0])
	row := env.repo.at("svc-1", clock(10, 39))
	require.NotNil(t, row)
	assert.EqualValues(t, 40, *row.Requests)
	assert.EqualValues(t, 3, *row.Errors)

	env.now = env.now.Add(time.Minute)
	env.sources.counts = nil
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
	assert.EqualValues(t, 40, *env.repo.at("svc-1", clock(10, 39)).Requests, "a minute the poll does not read again keeps its count")
	quiet := env.repo.at("svc-1", clock(10, 40))
	require.NotNil(t, quiet)
	require.NotNil(t, quiet.Requests, "an answered query with no traffic is zero requests")
	assert.Zero(t, *quiet.Requests)
}

func TestPollRequestCounts_AFailedQueryLeavesTheMinuteMissing(t *testing.T) {
	env := newRequestEnv(t, tunnelRoute)
	env.sources.zoneErr = map[string]error{"z1": errors.New("cloudflare down")}

	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	row := env.repo.at("svc-1", clock(10, 39))
	require.NotNil(t, row, "the gap is stored")
	assert.Nil(t, row.Requests, "a gap is never zero")
	assert.Nil(t, row.Errors)

	env.now = env.now.Add(time.Minute)
	env.sources.zoneErr = nil
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
	assert.Nil(t, env.repo.at("svc-1", clock(10, 39)).Requests, "the next poll does not go back for it")
	assert.NotNil(t, env.repo.at("svc-1", clock(10, 40)).Requests)
}

func TestPollRequestCounts_AStackOnBothGatewaysKeepsTheProxyCountWhenTheTunnelMinuteIsSet(t *testing.T) {
	env := newRequestEnv(t, proxyRoute, tunnelRoute)
	env.now = clock(10, 39).Add(20 * time.Second)
	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@file", Requests: 10, Errors: 2})
	env.now = clock(10, 42).Add(30 * time.Second)
	env.sources.counts = map[string][]HostMinute{"z1": {tunnelHost("app.example.com", clock(10, 39), 56, 4)}}

	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	row := env.repo.at("svc-1", clock(10, 39))
	require.NotNil(t, row)
	assert.EqualValues(t, 66, *row.Requests, "the tunnel's count is set beside the proxy's, not over it")
	assert.EqualValues(t, 6, *row.Errors)
}

func TestPollRequestCounts_AFailedTunnelReadLeavesTheProxyCountOfTheMinute(t *testing.T) {
	env := newRequestEnv(t, proxyRoute, tunnelRoute)
	env.now = clock(10, 39).Add(20 * time.Second)
	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@file", Requests: 10})
	env.now = clock(10, 42).Add(30 * time.Second)
	env.sources.zoneErr = map[string]error{"z1": errors.New("cloudflare down")}

	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	assert.EqualValues(t, 10, *env.repo.at("svc-1", clock(10, 39)).Requests, "one source counted it, so it is not missing")
	other := env.repo.at("svc-1", clock(10, 38))
	require.NotNil(t, other)
	assert.Nil(t, other.Requests, "no source counted it")
}

func tunnelHost(host string, minute time.Time, requests, errs int64) HostMinute {
	return HostMinute{Hostname: host, Minute: minute, Requests: requests, Errors: errs}
}

func TestPollRequestCounts_ReadingTheSameTunnelMinuteAgainLeavesTheCount(t *testing.T) {
	env := newRequestEnv(t, tunnelRoute)
	env.sources.counts = map[string][]HostMinute{"z1": {tunnelHost("app.example.com", clock(10, 39), 56, 4)}}
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	// a restart: a fresh service over the same stored minutes re-reads the recent ones.
	env.svc.SetRequestCounts(env.sources, env.repo)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
	env.now = env.now.Add(time.Minute)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	row := env.repo.at("svc-1", clock(10, 39))
	assert.EqualValues(t, 56, *row.Requests)
	assert.EqualValues(t, 4, *row.Errors)
}

func TestPollRequestCounts_ARestartReadsTheMinutesItSkipped(t *testing.T) {
	env := newRequestEnv(t, tunnelRoute)
	env.sources.counts = map[string][]HostMinute{"z1": {tunnelHost("app.example.com", clock(10, 39), 5, 0)}}
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	env.now = env.now.Add(3 * time.Minute)
	env.sources.counts = map[string][]HostMinute{"z1": {
		tunnelHost("app.example.com", clock(10, 39), 5, 0),
		tunnelHost("app.example.com", clock(10, 40), 6, 1),
		tunnelHost("app.example.com", clock(10, 41), 7, 0),
		tunnelHost("app.example.com", clock(10, 42), 8, 2),
	}}
	env.svc.SetRequestCounts(env.sources, env.repo)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	for m, want := range map[int]int64{39: 5, 40: 6, 41: 7, 42: 8} {
		row := env.repo.at("svc-1", clock(10, m))
		require.NotNil(t, row, "minute 10:%d", m)
		require.NotNil(t, row.Requests, "minute 10:%d", m)
		assert.Equal(t, want, *row.Requests, "minute 10:%d", m)
	}
}

func TestPollRequestCounts_HostnamesAndZonesOfOneStackCombineInAMinute(t *testing.T) {
	second := RequestRoute{Router: "nexul-e3", Hostname: "api.example.com", ContainerID: "c-web", ZoneID: "z1", Machine: outageMachine, Tunnel: true}
	other := RequestRoute{Router: "nexul-e4", Hostname: "app.example.org", ContainerID: "c-web", ZoneID: "z2", Machine: outageMachine, Tunnel: true}
	env := newRequestEnv(t, tunnelRoute, second, other)
	env.sources.counts = map[string][]HostMinute{
		"z1": {tunnelHost("app.example.com", clock(10, 39), 10, 1), tunnelHost("api.example.com", clock(10, 39), 20, 2)},
		"z2": {tunnelHost("app.example.org", clock(10, 39), 30, 3)},
	}

	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	row := env.repo.at("svc-1", clock(10, 39))
	assert.EqualValues(t, 60, *row.Requests)
	assert.EqualValues(t, 6, *row.Errors)
}

func TestPollRequestCounts_AFailedZoneLeavesTheStacksOtherZoneUnwritten(t *testing.T) {
	other := RequestRoute{Router: "nexul-e4", Hostname: "app.example.org", ContainerID: "c-web", ZoneID: "z2", Machine: outageMachine, Tunnel: true}
	env := newRequestEnv(t, tunnelRoute, other)
	env.sources.counts = map[string][]HostMinute{"z1": {tunnelHost("app.example.com", clock(10, 39), 10, 1)}}
	env.sources.zoneErr = map[string]error{"z2": errors.New("cloudflare down")}

	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	row := env.repo.at("svc-1", clock(10, 39))
	require.NotNil(t, row)
	assert.Nil(t, row.Requests, "half a sum is not a count")
}

func TestPollRequestCounts_ProxyFramesStillAddAcrossGateways(t *testing.T) {
	second := RequestRoute{Router: "nexul-e5", Hostname: "api.example.com", ContainerID: "c-web", Machine: "other-box"}
	env := newRequestEnv(t, proxyRoute, second)
	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@file", Requests: 10, Errors: 1})
	env.counts(t, "other-box", routerCount{Router: "nexul-e5@file", Requests: 5, Errors: 2})
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	row := env.repo.at("svc-1", clock(10, 42))
	assert.EqualValues(t, 15, *row.Requests)
	assert.EqualValues(t, 3, *row.Errors)
}

func TestPollRequestCounts_AProxyMinuteNoRunnerReportedIsMissingAndAReportedOneIsKept(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	env.now = clock(10, 39).Add(5 * time.Second)
	env.counts(t, outageMachine, routerCount{Router: "nexul-e1@file", Requests: 7})
	env.now = clock(10, 42).Add(30 * time.Second)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	assert.EqualValues(t, 7, *env.repo.at("svc-1", clock(10, 39)).Requests, "a reported minute is not overwritten")
	assert.Empty(t, env.sources.calls, "no Cloudflare query for a proxy stack")
}

func TestPollRequestCounts_CatchesUpAfterAStallButOnlyAFewMinutes(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
	env.now = env.now.Add(30 * time.Minute)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))

	// the second poll ends at 11:10 and marks the five minutes before it.
	for m := 5; m <= 9; m++ {
		row := env.repo.at("svc-1", clock(11, m))
		require.NotNil(t, row, "minute 11:0%d", m)
		assert.Nil(t, row.Requests)
	}
	assert.Nil(t, env.repo.at("svc-1", clock(11, 4)), "older minutes of the stall are not backfilled")
}

func TestPollRequestCounts_PrunesADayBack(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	require.NoError(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
	assert.Equal(t, clock(10, 40).Add(-24*time.Hour), env.repo.pruned)
}

func TestPollRequestCounts_StopsAtAFailedRouteList(t *testing.T) {
	env := newRequestEnv(t)
	env.sources.routesErr = errors.New("db down")
	assert.Error(t, env.svc.pollRequestCounts(t.Context(), slog.Default()))
}

func TestRequestWindow_PlacesMinutesAndKeepsGapsNull(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	now := env.now.Truncate(time.Minute)
	require.NoError(t, env.repo.Add(t.Context(), []RequestAdd{
		{StackID: "svc-1", Minute: now, Requests: 8, Errors: 2},
		{StackID: "svc-1", Minute: now.Add(-3 * time.Minute), Requests: 0},
		{StackID: "svc-1", Minute: now.Add(-2 * time.Hour), Requests: 99},
	}))
	require.NoError(t, env.repo.MarkMissing(t.Context(), SourceProxy, []string{"svc-1"}, now.Add(-time.Minute)))

	w, err := env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)

	assert.True(t, w.Counted)
	assert.Equal(t, now.Add(-59*time.Minute), w.From)
	require.Len(t, w.Requests, 60)
	assert.EqualValues(t, 8, *w.Requests[59])
	assert.EqualValues(t, 2, *w.Errors5xx[59])
	assert.Nil(t, w.Requests[58], "a marked-missing minute is null")
	assert.Nil(t, w.Requests[57], "a minute with no row is null")
	require.NotNil(t, w.Requests[56])
	assert.Zero(t, *w.Requests[56], "a counted quiet minute is zero")
}

func TestRequestWindow_IsNotCountedForAStackWithNoHostnameOrNoWiring(t *testing.T) {
	env := newRequestEnv(t)
	stack, err := env.svc.stacks.GetByID(t.Context(), "svc-1")
	require.NoError(t, err)
	w, err := env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.False(t, w.Counted)
	assert.Empty(t, w.Requests)

	env.sources.routes = []RequestRoute{proxyRoute}
	env.svc.requests = nil
	w, err = env.svc.RequestWindow(t.Context(), stack)
	require.NoError(t, err)
	assert.False(t, w.Counted)

	env.svc.SetRequestCounts(env.sources, env.repo)
	env.sources.routesErr = errors.New("db down")
	_, err = env.svc.RequestWindow(t.Context(), stack)
	assert.Error(t, err)
}

func TestRunRequestCounts_StopsWhenItsContextEnds(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		env.svc.RunRequestCounts(ctx, slog.Default())
		close(done)
	}()
	cancel()
	<-done
	assert.NotZero(t, env.repo.pruned, "it polled once before it waited")

	env.svc.requests = nil
	env.svc.RunRequestCounts(t.Context(), slog.Default())
}

func TestStackGet_CarriesTheLastHourOfRequestsForAStackWithAHostname(t *testing.T) {
	s, stack, _ := stackFixture(t)
	containers, err := s.ListServices(t.Context(), stack.ID)
	require.NoError(t, err)
	repo := newFakeRequestRepo()
	sources := &fakeRequestSources{routes: []RequestRoute{{Router: "nexul-e1", ContainerID: containers[0].ID, Machine: stack.Machine}}}
	s.SetRequestCounts(sources, repo)
	now := s.now().Truncate(time.Minute)
	require.NoError(t, repo.Add(t.Context(), []RequestAdd{
		{StackID: stack.ID, Minute: now, Requests: 30, Errors: 4},
		{StackID: stack.ID, Minute: now.Add(-time.Minute), Requests: 10, Errors: 1},
	}))

	got, err := call(t, s, "stack_get", `{"id":"`+stack.ID+`"}`)
	require.NoError(t, err)

	r := got.(stackDetail).Requests
	require.NotNil(t, r)
	assert.EqualValues(t, 40, r.TotalRequests)
	assert.EqualValues(t, 5, r.Total5xx)
	assert.Equal(t, 58, r.MissingMinutes)
	assert.Len(t, r.Requests, 60)
	assert.Contains(t, asJSON(t, got), `"requests_per_minute":[null,`)

	sources.routes = nil
	got, err = call(t, s, "stack_get", `{"id":"`+stack.ID+`"}`)
	require.NoError(t, err)
	assert.Nil(t, got.(stackDetail).Requests, "a stack with no hostname has no requests field")
	assert.NotContains(t, asJSON(t, got), "requests_per_minute")
}

func TestRequestsRoute_ServesTheWindowAndRefusesAMissingStack(t *testing.T) {
	env := newRequestEnv(t, proxyRoute)
	h := NewHandler(env.svc).Routes()
	now := env.now.Truncate(time.Minute)
	require.NoError(t, env.repo.Add(t.Context(), []RequestAdd{{StackID: "svc-1", Minute: now, Requests: 6, Errors: 1}}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/svc-1/requests", nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var w RequestWindow
	decodeDeploy(t, rec, &w)
	assert.True(t, w.Counted)
	require.Len(t, w.Requests, 60)
	assert.EqualValues(t, 6, *w.Requests[59])
	assert.Nil(t, w.Requests[0])

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/nope/requests", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	env.sources.routesErr = errors.New("db down")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stacks/svc-1/requests", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
