package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// TopicRunnerRequestCounts is the internal bus topic a machine runner's proxy counts arrive on; never part of the catalog.
const TopicRunnerRequestCounts = "runner.request_counts"

const (
	// requestRetention is how long a stack's minutes are kept.
	requestRetention = 24 * time.Hour
	// requestWindowMinutes is how many minutes a stack's request counts show.
	requestWindowMinutes = 60
	// requestLag keeps the tunnel poll behind now, because Cloudflare's dataset arrives a little late.
	requestLag = 2 * time.Minute
	// requestCatchUp bounds how many minutes one poll reads or marks after a stall; older ones stay missing.
	requestCatchUp = 5
	requestEvery   = time.Minute
)

// RequestRoute is one exposure as request counting sees it, mirrored from dns (ADR 0017): the Traefik router its
// labels name, the hostname and the container it reaches, and who counts it.
type RequestRoute struct {
	Router      string
	Hostname    string
	ContainerID string
	ZoneID      string
	Machine     string
	// Tunnel is true when Cloudflare counts the hostname and false when the machine's Traefik does.
	Tunnel bool
}

// HostMinute is one hostname's requests in one minute, and how many answered 5xx.
type HostMinute struct {
	Hostname string
	Minute   time.Time
	Requests int64
	Errors   int64
}

// RequestSources is the dns seam request counting reads (ADR 0017): where hostnames are routed, and Cloudflare's
// per-minute counts for the tunnel ones.
type RequestSources interface {
	Routes(ctx context.Context) ([]RequestRoute, error)
	TunnelCounts(ctx context.Context, zoneID string, hostnames []string, from, to time.Time) ([]HostMinute, error)
}

// RequestAdd is requests and 5xx answers to add to a stack's minute.
type RequestAdd struct {
	StackID  string
	Minute   time.Time
	Requests int64
	Errors   int64
}

// RequestMinute is one stored minute; nil counts mean nothing could count it, which is not zero.
type RequestMinute struct {
	Minute   time.Time
	Requests *int64
	Errors   *int64
}

// RequestRepo is the persistence contract for a stack's per-minute request counts.
type RequestRepo interface {
	// Add sums each entry into its minute, which replaces a missing one.
	Add(ctx context.Context, adds []RequestAdd) error
	// MarkMissing stores the minute as missing for each stack that has no row for it yet.
	MarkMissing(ctx context.Context, stackIDs []string, minute time.Time) error
	// Since returns a stack's stored minutes from the given one on, oldest first.
	Since(ctx context.Context, stackID string, from time.Time) ([]RequestMinute, error)
	// Prune drops every stack's minutes before the given time.
	Prune(ctx context.Context, before time.Time) error
}

// RequestWindow is a stack's last hour, one slot per minute from From; null is a minute nobody could count.
type RequestWindow struct {
	Counted   bool      `json:"counted"`
	From      time.Time `json:"from"`
	Requests  []*int64  `json:"requests"`
	Errors5xx []*int64  `json:"errors_5xx"`
}

// requestState is the counting wiring and the poll's place in time.
type requestState struct {
	sources RequestSources
	repo    RequestRepo
	// polled is the end of the last window read; the next one starts there.
	polled time.Time
}

// routerCount and requestCountsEvent mirror the runner's event independently (ADR 0017).
type routerCount struct {
	Router   string `json:"router"`
	Requests int64  `json:"requests"`
	Errors   int64  `json:"errors"`
}

type requestCountsEvent struct {
	Machine string        `json:"machine"`
	Counts  []routerCount `json:"counts"`
}

// SetRequestCounts wires request counting; unset, no stack has counts.
func (s *Service) SetRequestCounts(sources RequestSources, repo RequestRepo) {
	s.requests = &requestState{sources: sources, repo: repo}
}

// wantsCounts says whether a stack's requests are worth counting. Every stack with a hostname is for now; the
// error-rate rule narrows this to the stacks that have it turned on.
func wantsCounts(*Stack) bool { return true }

// HandleRequestCounts is the runner.request_counts consumer: it adds what a machine's proxy routers served to the
// minute it arrived in. Every stack the machine's proxy gateways carry gets its minute, even at zero requests, since
// a frame that arrived says the counters were read.
func (s *Service) HandleRequestCounts(ctx context.Context, ev eventbus.Event) error {
	var p requestCountsEvent
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return apperrs.Fatal(fmt.Errorf("parse %s: %w", TopicRunnerRequestCounts, err))
	}
	if s.requests == nil {
		return nil
	}
	routes, err := s.requests.sources.Routes(ctx)
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}
	routerStack := map[string]string{}
	minute := s.now().UTC().Truncate(time.Minute)
	adds := map[string]*RequestAdd{}
	for _, r := range routes {
		if r.Tunnel || r.Machine != p.Machine {
			continue
		}
		stackID, err := s.countedStack(ctx, r.ContainerID)
		if err != nil {
			return err
		}
		if stackID == "" {
			continue
		}
		routerStack[r.Router] = stackID
		adds[stackID] = &RequestAdd{StackID: stackID, Minute: minute}
	}
	for _, c := range p.Counts {
		router, _, _ := strings.Cut(c.Router, "@")
		if add := adds[routerStack[router]]; add != nil {
			add.Requests += c.Requests
			add.Errors += c.Errors
		}
	}
	return s.requests.repo.Add(ctx, addList(adds))
}

// countedStack is the stack a container belongs to when its requests are counted, "" otherwise.
func (s *Service) countedStack(ctx context.Context, containerID string) (string, error) {
	c, err := s.services.Get(ctx, containerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get container %s: %w", containerID, err)
	}
	stack, err := s.stacks.GetByID(ctx, c.StackID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get stack %s: %w", c.StackID, err)
	}
	if !wantsCounts(stack) {
		return "", nil
	}
	return stack.ID, nil
}

// RunRequestCounts polls Cloudflare for the tunnel hostnames' counts every minute, marks the minutes nobody counted
// as missing and drops what is older than a day.
func (s *Service) RunRequestCounts(ctx context.Context, log *slog.Logger) {
	if s.requests == nil {
		return
	}
	ticker := time.NewTicker(requestEvery)
	defer ticker.Stop()
	for {
		if err := s.pollRequestCounts(ctx, log); err != nil && ctx.Err() == nil {
			log.Warn("request counts poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// pollRequestCounts reads the minutes that have settled since the last poll: tunnel hostnames from Cloudflare, one
// query per zone. A zone that fails leaves its minutes unwritten, and so missing, rather than retrying into a later
// minute. Then every counted stack's still-empty minute is marked missing.
func (s *Service) pollRequestCounts(ctx context.Context, log *slog.Logger) error {
	end := s.now().UTC().Add(-requestLag).Truncate(time.Minute)
	from := s.requests.polled
	if from.Before(end.Add(-requestCatchUp * time.Minute)) {
		from = end.Add(-requestCatchUp * time.Minute)
	}
	if s.requests.polled.IsZero() {
		from = end.Add(-time.Minute)
	}
	defer func() { s.requests.polled = end }()
	if err := s.requests.repo.Prune(ctx, end.Add(-requestRetention)); err != nil {
		return fmt.Errorf("prune request counts: %w", err)
	}
	if !from.Before(end) {
		return nil
	}
	routes, err := s.requests.sources.Routes(ctx)
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}
	counted := map[string]bool{}
	byZone := map[string][]RequestRoute{}
	stackOf := map[string]string{}
	for _, r := range routes {
		stackID, err := s.countedStack(ctx, r.ContainerID)
		if err != nil {
			return err
		}
		if stackID == "" {
			continue
		}
		counted[stackID] = true
		stackOf[r.ContainerID] = stackID
		if r.Tunnel {
			byZone[r.ZoneID] = append(byZone[r.ZoneID], r)
		}
	}
	for zone, zoneRoutes := range byZone {
		adds, err := s.tunnelAdds(ctx, zone, zoneRoutes, stackOf, from, end)
		if err != nil {
			log.Warn("request counts unavailable for a zone", "zone", zone, "error", err)
			continue
		}
		if err := s.requests.repo.Add(ctx, adds); err != nil {
			return fmt.Errorf("add request counts: %w", err)
		}
	}
	for m := from; m.Before(end); m = m.Add(time.Minute) {
		if err := s.requests.repo.MarkMissing(ctx, slices.Sorted(maps.Keys(counted)), m); err != nil {
			return fmt.Errorf("mark missing request minutes: %w", err)
		}
	}
	return nil
}

// tunnelAdds reads a zone's hostnames over [from, end) and returns a row for each stack and minute, at zero where
// Cloudflare reported nothing: a successful read with no traffic is zero requests, not a gap.
func (s *Service) tunnelAdds(ctx context.Context, zone string, routes []RequestRoute, stackOf map[string]string, from, end time.Time) ([]RequestAdd, error) {
	stackByHost := map[string]string{}
	for _, r := range routes {
		stackByHost[strings.ToLower(r.Hostname)] = stackOf[r.ContainerID]
	}
	counts, err := s.requests.sources.TunnelCounts(ctx, zone, slices.Sorted(maps.Keys(stackByHost)), from, end)
	if err != nil {
		return nil, err
	}
	type key struct {
		stack  string
		minute time.Time
	}
	sums := map[key]*RequestAdd{}
	for _, stackID := range stackByHost {
		for m := from; m.Before(end); m = m.Add(time.Minute) {
			sums[key{stackID, m}] = &RequestAdd{StackID: stackID, Minute: m}
		}
	}
	for _, c := range counts {
		if add := sums[key{stackByHost[strings.ToLower(c.Hostname)], c.Minute.UTC()}]; add != nil {
			add.Requests += c.Requests
			add.Errors += c.Errors
		}
	}
	return addList(sums), nil
}

// RequestWindow returns a stack's last hour of counts. The caller has already read the stack. Counted is false for a
// stack with no hostname, which has nothing to count.
func (s *Service) RequestWindow(ctx context.Context, stack *Stack) (*RequestWindow, error) {
	w := &RequestWindow{Requests: []*int64{}, Errors5xx: []*int64{}}
	if s.requests == nil || !wantsCounts(stack) {
		return w, nil
	}
	counted, err := s.hasRoute(ctx, stack.ID)
	if err != nil || !counted {
		return w, err
	}
	w.Counted = true
	w.From = s.now().UTC().Truncate(time.Minute).Add(-(requestWindowMinutes - 1) * time.Minute)
	w.Requests = make([]*int64, requestWindowMinutes)
	w.Errors5xx = make([]*int64, requestWindowMinutes)
	rows, err := s.requests.repo.Since(ctx, stack.ID, w.From)
	if err != nil {
		return nil, fmt.Errorf("read request counts of stack %s: %w", stack.ID, err)
	}
	for _, row := range rows {
		if i := int(row.Minute.Sub(w.From) / time.Minute); i >= 0 && i < requestWindowMinutes {
			w.Requests[i], w.Errors5xx[i] = row.Requests, row.Errors
		}
	}
	return w, nil
}

// hasRoute reports whether any exposure reaches one of the stack's containers.
func (s *Service) hasRoute(ctx context.Context, stackID string) (bool, error) {
	routes, err := s.requests.sources.Routes(ctx)
	if err != nil {
		return false, fmt.Errorf("list routes: %w", err)
	}
	containers, err := s.services.ListByStack(ctx, stackID)
	if err != nil {
		return false, fmt.Errorf("list containers of stack %s: %w", stackID, err)
	}
	ids := map[string]bool{}
	for _, c := range containers {
		ids[c.ID] = true
	}
	return slices.ContainsFunc(routes, func(r RequestRoute) bool { return ids[r.ContainerID] }), nil
}

// addList flattens adds in a stable order.
func addList[K comparable](adds map[K]*RequestAdd) []RequestAdd {
	list := make([]RequestAdd, 0, len(adds))
	for _, a := range adds {
		list = append(list, *a)
	}
	slices.SortFunc(list, func(a, b RequestAdd) int {
		if c := strings.Compare(a.StackID, b.StackID); c != 0 {
			return c
		}
		return a.Minute.Compare(b.Minute)
	})
	return list
}
