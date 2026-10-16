package runner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const metricsPage = `# HELP traefik_router_requests_total How many HTTP requests are processed on a router.
# TYPE traefik_router_requests_total counter
traefik_router_requests_total{code="200",method="GET",protocol="http",router="nexul-e1@docker",service="app@docker"} 90
traefik_router_requests_total{code="502",method="GET",protocol="http",router="nexul-e1@docker",service="app@docker"} 4
traefik_router_requests_total{code="503",method="POST",protocol="http",router="nexul-e1@docker",service="app@docker"} 1
traefik_router_requests_total{code="200",method="GET",protocol="http",router="nexul@file",service="nexul@file"} 7
traefik_service_requests_total{code="200",service="app@docker"} 99
traefik_router_requests_total{code="404",router=""} 3
traefik_router_requests_total{code="200",router="broken"} not-a-number
`

func TestParseRouterCounters_ReadsTheRouterCounterAndNothingElse(t *testing.T) {
	got := parseRouterCounters("gw", metricsPage)

	assert.Equal(t, map[counterKey]int64{
		{"gw", "nexul-e1@docker", "200"}: 90,
		{"gw", "nexul-e1@docker", "502"}: 4,
		{"gw", "nexul-e1@docker", "503"}: 1,
		{"gw", "nexul@file", "200"}:      7,
	}, got, "service counters, a series with no router and an unreadable value are skipped")
}

func TestRequestReader_SendsChangesSinceTheLastRead(t *testing.T) {
	series := func(ok, bad int64) map[counterKey]int64 {
		return map[counterKey]int64{{"gw", "r1", "200"}: ok, {"gw", "r1", "502"}: bad}
	}
	r := newRequestReader()

	assert.Empty(t, r.deltas("gw", series(1000, 50)), "the first read of a gateway only sets the baseline")
	assert.Equal(t, []RouterCount{{Router: "r1", Requests: 15, Errors: 5}}, r.deltas("gw", series(1010, 55)))
	assert.Empty(t, r.deltas("gw", series(1010, 55)), "a quiet router adds nothing")
	assert.Equal(t, []RouterCount{{Router: "r1", Requests: 3, Errors: 1}}, r.deltas("gw", series(2, 1)),
		"a counter that went down was reset by a restart, so it counts from zero")
}

func TestRequestReader_ASeriesBornAfterTheBaselineCountsFromZeroAndGatewaysStayApart(t *testing.T) {
	r := newRequestReader()
	r.deltas("gw", map[counterKey]int64{{"gw", "r1", "200"}: 5})

	got := r.deltas("gw", map[counterKey]int64{{"gw", "r1", "200"}: 5, {"gw", "r2", "500"}: 2})
	assert.Equal(t, []RouterCount{{Router: "r2", Requests: 2, Errors: 2}}, got)

	assert.Empty(t, r.deltas("other", map[counterKey]int64{{"other", "r1", "200"}: 800}), "another gateway has its own baseline")
}

// fakeTraefikHost stands in for the host's Docker and the gateways' metrics pages.
type fakeTraefikHost struct {
	mu        sync.Mutex
	ps        string
	addresses map[string]string
	pages     map[string][]string
	fetchErr  error
	urls      []string
}

func (h *fakeTraefikHost) watch() *ContainerWatch {
	return &ContainerWatch{
		Stream: func(ctx context.Context, _, _ io.Writer, _ string, _ ...string) error { <-ctx.Done(); return ctx.Err() },
		Output: h.output,
		Fetch:  h.fetch,
	}
}

func (h *fakeTraefikHost) output(_ context.Context, args ...string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if args[0] == "ps" {
		return h.ps, nil
	}
	if args[0] == "inspect" && len(args) == 4 {
		return h.addresses[args[3]], nil
	}
	return "", nil
}

func (h *fakeTraefikHost) fetch(_ context.Context, url string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.urls = append(h.urls, url)
	if h.fetchErr != nil {
		return "", h.fetchErr
	}
	pages := h.pages[url]
	if len(pages) == 0 {
		return "", errors.New("connection refused")
	}
	page := pages[0]
	if len(pages) > 1 {
		h.pages[url] = pages[1:]
	}
	return page, nil
}

func (h *fakeTraefikHost) failWith(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fetchErr = err
}

func countsClient(t *testing.T, h *fakeTraefikHost) <-chan Frame {
	t.Helper()
	frames := make(chan Frame, 64)
	srv := wsTestServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for {
			var f Frame
			if err := wsjson.Read(ctx, conn, &f); err != nil {
				return
			}
			if f.Type == FrameRequestCounts {
				frames <- f
			}
		}
	})
	c := newTestClient(wsURL(srv), &fakeExecutor{})
	c.cfg.Containers, c.requestEvery = h.watch(), 10*time.Millisecond
	c.findT3 = func(string) string { return "" }
	runClient(t, c)
	return frames
}

func page(ok, bad int) string {
	return strings.NewReplacer("OK", strconv.Itoa(ok), "BAD", strconv.Itoa(bad)).Replace(
		`traefik_router_requests_total{code="200",router="nexul-e1@docker"} OK` + "\n" +
			`traefik_router_requests_total{code="502",router="nexul-e1@docker"} BAD` + "\n")
}

func TestWatchRequests_ReadsTheGatewayOnItsAddressAndSendsDeltas(t *testing.T) {
	h := &fakeTraefikHost{
		ps:        "gw1 traefik:v3\ndb1 postgres:16\nweb1 docker.io/library/traefik:v3.1\nbad\n",
		addresses: map[string]string{"gw1": "172.20.0.2 172.21.0.2 \n", "web1": "\n"},
		pages:     map[string][]string{"http://172.20.0.2:8082/metrics": {page(100, 10), page(130, 12)}},
	}
	frames := countsClient(t, h)

	first := nextFrame(t, frames)
	assert.Empty(t, first.Counts, "the first read is the baseline, and the frame still says the read worked")
	second := nextFrame(t, frames)
	assert.Equal(t, []RouterCount{{Router: "nexul-e1@docker", Requests: 32, Errors: 2}}, second.Counts)
	h.mu.Lock()
	defer h.mu.Unlock()
	assert.NotContains(t, strings.Join(h.urls, " "), "postgres")
}

func TestWatchRequests_ARead_ThatFailsSendsNothing(t *testing.T) {
	h := &fakeTraefikHost{
		ps:        "gw1 traefik:v3\n",
		addresses: map[string]string{"gw1": "172.20.0.2 \n"},
		pages:     map[string][]string{"http://172.20.0.2:8082/metrics": {page(1, 0)}},
	}
	h.failWith(errors.New("metrics are not on"))
	frames := countsClient(t, h)

	select {
	case f := <-frames:
		t.Fatalf("unexpected frame %+v", f)
	case <-time.After(150 * time.Millisecond):
	}

	h.failWith(nil)
	assert.Empty(t, nextFrame(t, frames).Counts, "the first successful read is a baseline")
}

func TestWatchRequests_AHostWithoutAGatewayOrWithoutDockerSendsNothing(t *testing.T) {
	h := &fakeTraefikHost{ps: "db1 postgres:16\n"}
	frames := countsClient(t, h)
	select {
	case f := <-frames:
		t.Fatalf("unexpected frame %+v", f)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestHostMetricsFetch_ReadsAPageAndRefusesAnErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			_, _ = io.WriteString(w, "traefik_router_requests_total{code=\"200\",router=\"r\"} 1\n")
			return
		}
		http.Error(w, "no", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	body, err := hostMetricsFetch(t.Context(), srv.URL+"/metrics")
	require.NoError(t, err)
	assert.Contains(t, body, "traefik_router_requests_total")

	_, err = hostMetricsFetch(t.Context(), srv.URL+"/other")
	assert.ErrorContains(t, err, "status 404")
	_, err = hostMetricsFetch(t.Context(), "http://127.0.0.1:1/metrics")
	assert.Error(t, err)
}

func TestHost_RequestCountsBecomeAnInternalBusEventStampedWithTheMachine(t *testing.T) {
	f := newT3Fixture(t)
	_, deploy := f.control(t, f.deploy)
	_, alice := f.control(t, f.alice)
	ctx := t.Context()
	counts := []RouterCount{{Router: "nexul-e1@docker", Requests: 9, Errors: 2}}

	require.NoError(t, wsjson.Write(ctx, alice, Frame{Type: FrameRequestCounts, Counts: counts}))
	require.NoError(t, wsjson.Write(ctx, deploy, Frame{Type: FrameRequestCounts, Counts: counts}))

	eventually(t, 2*time.Second, func() bool { return len(f.bus.topicEvents(TopicRequestCounts)) > 0 })
	require.Len(t, f.bus.topicEvents(TopicRequestCounts), 1, "a personal runner's counts are dropped")
	var ev RequestCountsEvent
	require.NoError(t, json.Unmarshal(f.bus.topicEvents(TopicRequestCounts)[0].Payload, &ev))
	assert.Equal(t, RequestCountsEvent{Machine: "m1", Counts: counts}, ev, "the machine is the runner's own, never the frame's")
}
