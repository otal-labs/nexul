package runner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Where a proxy gateway's Traefik serves its router counters, and how often a machine runner reads them.
const (
	traefikMetricsPort   = "8082"
	traefikRouterCounter = "traefik_router_requests_total"
	requestCountsEvery   = 30 * time.Second
	metricsFetchWait     = 10 * time.Second
	metricsMaxBytes      = 8 << 20
	gatewayImageRepo     = "traefik"
	psGatewaysFormat     = "{{.ID}} {{.Image}}"
	inspectAddressesFmt  = `{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}`
)

// RouterCount is one router's requests since the previous read, and how many of them answered 5xx.
type RouterCount struct {
	Router   string `json:"router"`
	Requests int64  `json:"requests"`
	Errors   int64  `json:"errors"`
}

// counterKey names one Prometheus series, so a reset is judged per series and not on a router's sum.
type counterKey struct {
	gateway string
	router  string
	code    string
}

// requestReader turns successive reads of Traefik's cumulative counters into the counts since the last read.
type requestReader struct {
	last map[counterKey]int64
	// seen holds the gateways already read once: a first read only sets the baseline, since a counter that has run for
	// days would otherwise land in a single minute.
	seen map[string]bool
}

func newRequestReader() *requestReader {
	return &requestReader{last: map[counterKey]int64{}, seen: map[string]bool{}}
}

// deltas returns each router's change since the previous read of gateway; a series that went down was reset by a
// restart and counts from zero.
func (r *requestReader) deltas(gateway string, series map[counterKey]int64) []RouterCount {
	first := !r.seen[gateway]
	r.seen[gateway] = true
	byRouter := map[string]*RouterCount{}
	var order []string
	for k, cur := range series {
		prev, known := r.last[k]
		r.last[k] = cur
		if first {
			continue
		}
		d := cur - prev
		if !known || cur < prev {
			d = cur
		}
		if d <= 0 {
			continue
		}
		rc := byRouter[k.router]
		if rc == nil {
			rc = &RouterCount{Router: k.router}
			byRouter[k.router] = rc
			order = append(order, k.router)
		}
		rc.Requests += d
		if strings.HasPrefix(k.code, "5") {
			rc.Errors += d
		}
	}
	counts := make([]RouterCount, 0, len(order))
	for _, router := range order {
		counts = append(counts, *byRouter[router])
	}
	return counts
}

// parseRouterCounters reads traefik_router_requests_total{code,router} out of a Prometheus text exposition.
func parseRouterCounters(gateway, text string) map[counterKey]int64 {
	series := map[counterKey]int64{}
	for line := range strings.Lines(text) {
		rest, ok := strings.CutPrefix(line, traefikRouterCounter+"{")
		if !ok {
			continue
		}
		labels, value, ok := strings.Cut(rest, "} ")
		if !ok {
			continue
		}
		n, err := strconv.ParseFloat(strings.Fields(value)[0], 64)
		if err != nil || n < 0 {
			continue
		}
		code, router := labelValue(labels, "code"), labelValue(labels, "router")
		if router == "" {
			continue
		}
		series[counterKey{gateway: gateway, router: router, code: code}] += int64(n)
	}
	return series
}

// labelValue finds name="value" in a label list; router and code values hold no quotes or commas.
func labelValue(labels, name string) string {
	for part := range strings.SplitSeq(labels, ",") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(part), name+`="`); ok {
			return strings.TrimSuffix(v, `"`)
		}
	}
	return ""
}

// hostMetricsFetch GETs a metrics page from a container address on this host.
func hostMetricsFetch(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, metricsFetchWait)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, metricsMaxBytes))
	return string(body), err
}

// watchRequests reads the host's proxy gateways every requestCountsEvery and sends what changed. A read that fails
// sends nothing, which is how the server learns the minute is missing.
func (c *Client) watchRequests(ctx context.Context, conn *websocket.Conn) {
	reader := newRequestReader()
	ticker := time.NewTicker(c.requestEvery)
	defer ticker.Stop()
	for {
		c.sendRequestCounts(ctx, conn, reader)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Client) sendRequestCounts(ctx context.Context, conn *websocket.Conn, reader *requestReader) {
	counts, ok := c.readRequestCounts(ctx, reader)
	if !ok {
		return
	}
	c.sendFrame(ctx, conn, Frame{Type: FrameRequestCounts, Counts: counts})
}

// readRequestCounts reads every Traefik on the host that answers; ok is false when none did.
func (c *Client) readRequestCounts(ctx context.Context, reader *requestReader) (counts []RouterCount, ok bool) {
	gateways, err := c.gatewayContainers(ctx)
	if err != nil {
		c.log.Debug("traefik metrics unavailable", "runner", c.cfg.Name, "error", err)
		return nil, false
	}
	for _, id := range gateways {
		text, err := c.fetchGatewayMetrics(ctx, id)
		if err != nil {
			c.log.Debug("traefik metrics unavailable", "runner", c.cfg.Name, "container", id, "error", err)
			continue
		}
		ok = true
		counts = append(counts, reader.deltas(id, parseRouterCounters(id, text))...)
	}
	return counts, ok
}

// gatewayContainers lists the running containers whose image is Traefik.
func (c *Client) gatewayContainers(ctx context.Context) ([]string, error) {
	out, err := c.cfg.Containers.Output(ctx, "ps", "--no-trunc", "--format", psGatewaysFormat)
	if err != nil {
		return nil, err
	}
	var ids []string
	for line := range strings.Lines(out) {
		id, image, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		repo, _, _ := strings.Cut(image[strings.LastIndex(image, "/")+1:], ":")
		if repo == gatewayImageRepo {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// fetchGatewayMetrics reads the metrics page on the first of the container's addresses that answers.
func (c *Client) fetchGatewayMetrics(ctx context.Context, id string) (string, error) {
	out, err := c.cfg.Containers.Output(ctx, "inspect", "--format", inspectAddressesFmt, id)
	if err != nil {
		return "", err
	}
	lastErr := fmt.Errorf("container %s has no address", id)
	for addr := range strings.FieldsSeq(out) {
		text, err := c.cfg.Containers.Fetch(ctx, "http://"+net.JoinHostPort(addr, traefikMetricsPort)+"/metrics")
		if err == nil {
			return text, nil
		}
		lastErr = err
	}
	return "", lastErr
}
