package deploy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

const (
	// errorRateRecoveredFor is how long after an episode ends the stack page still reads "recovered".
	errorRateRecoveredFor = time.Hour
	// analyticsProbeTTL is how long a read of whether Cloudflare lets the token see a zone's analytics is reused.
	analyticsProbeTTL = time.Minute
	// errorRateSubject keys an episode's crash ticket beside the service names the same table holds; a service name
	// never contains a colon.
	errorRateSubject = "episode:error-rate"
)

// Why the rule cannot watch a stack; the switch greys out with this line.
const (
	ReasonNoHostname  = "Needs a hostname on this stack"
	ReasonNoAnalytics = "Your Cloudflare token can't read analytics; add Zone Analytics Read to it"
)

// ErrorRateEpisode is one stretch of a stack answering too many requests with 5xx: from a window crossing its rule
// until every minute of a window has stayed under the limit. Peak, requests and errors are those of the worst window.
type ErrorRateEpisode struct {
	ID            string     `json:"id"`
	StackID       string     `json:"stack_id"`
	StartedAt     time.Time  `json:"started_at"`
	PeakPercent   float64    `json:"peak_percent"`
	Requests      int64      `json:"requests"`
	Errors        int64      `json:"errors"`
	LimitPercent  float64    `json:"limit_percent"`
	WindowMinutes int        `json:"window_minutes"`
	MinRequests   int        `json:"min_requests"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
}

// ErrorRateEvent is the stack.error_rate_crossed and stack.error_rate_recovered payload. Like an outage it carries no
// request lines, only the counts of the worst window.
type ErrorRateEvent struct {
	WorkspaceID   string     `json:"workspace_id,omitempty" jsonschema:"The workspace of the stack's project; empty for an instance stack."`
	ProjectID     string     `json:"project_id,omitempty" jsonschema:"The stack's project; empty for an instance stack."`
	StackID       string     `json:"stack_id" jsonschema:"The stack whose answers failed."`
	StackName     string     `json:"stack_name" jsonschema:"The stack's name."`
	EpisodeID     string     `json:"episode_id" jsonschema:"The episode this event opens or closes."`
	PeakPercent   float64    `json:"peak_percent" jsonschema:"The highest share of 5xx answers, in percent, any window of the episode reached; with stack.error_rate_crossed, the share that tripped the rule."`
	Requests      int64      `json:"requests" jsonschema:"The requests in that worst window."`
	Errors        int64      `json:"errors" jsonschema:"The 5xx answers in that worst window."`
	LimitPercent  float64    `json:"limit_percent" jsonschema:"The limit, in percent, the rule held when the episode opened."`
	WindowMinutes int        `json:"window_minutes" jsonschema:"The minutes the rule read the share over when the episode opened."`
	MinRequests   int        `json:"min_requests" jsonschema:"The request floor the rule held when the episode opened."`
	StartedAt     time.Time  `json:"started_at" jsonschema:"When the rule tripped."`
	EndedAt       *time.Time `json:"ended_at,omitempty" jsonschema:"When the share was back under the limit; sent only with stack.error_rate_recovered."`
}

// ErrorRateRepo is the persistence contract for error-rate episodes; opening and closing write their events atomically.
type ErrorRateRepo interface {
	// Open inserts an episode and writes evts in the same transaction; a stack has at most one open episode.
	Open(ctx context.Context, e *ErrorRateEpisode, evts ...eventbus.OutboxEvent) error
	// UpdatePeak records a worse window on an open episode.
	UpdatePeak(ctx context.Context, e *ErrorRateEpisode) error
	// Close ends an open episode and writes evts in the same transaction.
	Close(ctx context.Context, id string, endedAt time.Time, evts ...eventbus.OutboxEvent) error
	// GetOpen returns a stack's open episode, ErrNotFound when it has none.
	GetOpen(ctx context.Context, stackID string) (*ErrorRateEpisode, error)
	// LatestEnd is when the stack's newest closed episode ended, the zero time when it has had none.
	LatestEnd(ctx context.Context, stackID string) (time.Time, error)
}

// SetErrorRateRepo wires episode storage; unset, no rule is evaluated and no stack has episodes.
func (s *Service) SetErrorRateRepo(r ErrorRateRepo) { s.errorRates = r }

// ErrorRateState is the stack's open episode, else when the last one ended if that was within the hour. A stack with
// the rule off has neither.
func (s *Service) ErrorRateState(ctx context.Context, stack *Stack) (*ErrorRateEpisode, *time.Time, error) {
	if s.errorRates == nil || !stack.ErrorRate.Enabled {
		return nil, nil, nil
	}
	open, err := s.errorRates.GetOpen(ctx, stack.ID)
	if err == nil {
		return open, nil, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil, fmt.Errorf("get open error-rate episode of stack %s: %w", stack.ID, err)
	}
	end, err := s.errorRates.LatestEnd(ctx, stack.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get last error-rate end of stack %s: %w", stack.ID, err)
	}
	if end.IsZero() || s.now().Sub(end) > errorRateRecoveredFor {
		return nil, nil, nil
	}
	return nil, &end, nil
}

// changeErrorRate settles a stack's rule changing from before to after: switching it on needs a stack it can watch, and
// switching it off ends an open episode.
func (s *Service) changeErrorRate(ctx context.Context, before, after *Stack) error {
	if err := s.requireErrorRateAvailable(ctx, before, after); err != nil {
		return err
	}
	return s.endErrorRateWhenOff(ctx, before, after)
}

// endErrorRateWhenOff ends the open episode of a stack whose rule was just switched off, with no recovered event: the
// share did not recover, nothing is watching it any more.
func (s *Service) endErrorRateWhenOff(ctx context.Context, before, after *Stack) error {
	if s.errorRates == nil || !before.ErrorRate.Enabled || after.ErrorRate.Enabled {
		return nil
	}
	open, err := s.errorRates.GetOpen(ctx, after.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get open error-rate episode of stack %s: %w", after.ID, err)
	}
	if err := s.errorRates.Close(ctx, open.ID, s.now().UTC()); err != nil && !errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("end error-rate episode %s: %w", open.ID, err)
	}
	return nil
}

// errorWindow is a rule's window of a stack's minutes: the counts summed, and whether every minute was counted.
type errorWindow struct {
	requests, errors int64
	complete         bool
	// minutes holds each minute's counts oldest first, nil where nobody could count it.
	minutes []RequestMinute
}

// readErrorWindow reads the rule's minutes ending before end. The window ends two minutes back, where the tunnel's
// counts have settled, so a proxy-counted stack is read behind the clock by the same amount.
func (s *Service) readErrorWindow(ctx context.Context, stackID string, rule ErrorRateRule, end time.Time) (errorWindow, error) {
	from := end.Add(-time.Duration(rule.Minutes) * time.Minute)
	rows, err := s.requests.repo.Since(ctx, stackID, from)
	if err != nil {
		return errorWindow{}, fmt.Errorf("read request counts of stack %s: %w", stackID, err)
	}
	w := errorWindow{complete: true, minutes: make([]RequestMinute, rule.Minutes)}
	for _, row := range rows {
		if i := int(row.Minute.Sub(from) / time.Minute); i >= 0 && i < rule.Minutes {
			w.minutes[i] = row
		}
	}
	for _, m := range w.minutes {
		if m.Requests == nil || m.Errors == nil {
			w.complete = false
			continue
		}
		w.requests += *m.Requests
		w.errors += *m.Errors
	}
	return w, nil
}

// overLimit says whether errors out of requests is a share above limit percent; no requests is no share.
func overLimit(errs, requests int64, limit float64) bool {
	return requests > 0 && float64(errs)*100 > limit*float64(requests)
}

func sharePercent(errs, requests int64) float64 {
	if requests == 0 {
		return 0
	}
	return float64(errs) * 100 / float64(requests)
}

// stayedUnder says whether every minute of the window was counted and at or under the limit; a minute with no
// requests is under it.
func (w errorWindow) stayedUnder(limit float64) bool {
	if !w.complete {
		return false
	}
	for _, m := range w.minutes {
		if overLimit(*m.Errors, *m.Requests, limit) {
			return false
		}
	}
	return true
}

// evaluateErrorRates checks the rule of every stack that has it on. One stack failing is logged and never stops the
// others.
func (s *Service) evaluateErrorRates(ctx context.Context, log *slog.Logger) {
	if s.errorRates == nil {
		return
	}
	stacks, err := s.stacks.ListWithErrorRate(ctx)
	if err != nil {
		log.Warn("list stacks with an error-rate rule failed", "error", err)
		return
	}
	end := s.now().UTC().Add(-requestLag).Truncate(time.Minute)
	for _, stack := range stacks {
		if err := s.evaluateErrorRate(ctx, stack, end); err != nil && ctx.Err() == nil {
			log.Warn("error-rate evaluation failed", "stack_id", stack.ID, "error", err)
		}
	}
}

// evaluateErrorRate opens an episode when the window crosses the rule and ends one when every minute of it has stayed
// under the limit. A window with a minute nobody counted does neither.
func (s *Service) evaluateErrorRate(ctx context.Context, stack *Stack, end time.Time) error {
	rule := stack.ErrorRate
	w, err := s.readErrorWindow(ctx, stack.ID, rule, end)
	if err != nil {
		return err
	}
	open, err := s.errorRates.GetOpen(ctx, stack.ID)
	if errors.Is(err, apperrs.ErrNotFound) {
		if !w.complete || w.requests < int64(rule.MinRequests) || !overLimit(w.errors, w.requests, rule.Percent) {
			return nil
		}
		return s.openErrorRate(ctx, stack, w)
	}
	if err != nil {
		return fmt.Errorf("get open error-rate episode of stack %s: %w", stack.ID, err)
	}
	if w.stayedUnder(rule.Percent) {
		return s.closeErrorRate(ctx, stack, open)
	}
	return s.raisePeak(ctx, open, w)
}

func (s *Service) openErrorRate(ctx context.Context, stack *Stack, w errorWindow) error {
	rule := stack.ErrorRate
	e := &ErrorRateEpisode{
		ID: ids.New(), StackID: stack.ID, StartedAt: s.now().UTC(), PeakPercent: sharePercent(w.errors, w.requests),
		Requests: w.requests, Errors: w.errors, LimitPercent: rule.Percent, WindowMinutes: rule.Minutes, MinRequests: rule.MinRequests,
	}
	evt, err := s.errorRateEvent(ctx, TopicErrorRateCrossed, stack, e)
	if err != nil {
		return err
	}
	if err := s.errorRates.Open(ctx, e, evt); err != nil {
		return fmt.Errorf("open error-rate episode of stack %s: %w", stack.ID, err)
	}
	return nil
}

// raisePeak keeps the worst complete window the open episode has seen.
func (s *Service) raisePeak(ctx context.Context, open *ErrorRateEpisode, w errorWindow) error {
	share := sharePercent(w.errors, w.requests)
	if !w.complete || share <= open.PeakPercent {
		return nil
	}
	open.PeakPercent, open.Requests, open.Errors = share, w.requests, w.errors
	if err := s.errorRates.UpdatePeak(ctx, open); err != nil {
		return fmt.Errorf("update error-rate episode %s: %w", open.ID, err)
	}
	return nil
}

func (s *Service) closeErrorRate(ctx context.Context, stack *Stack, open *ErrorRateEpisode) error {
	end := s.now().UTC()
	open.EndedAt = &end
	evt, err := s.errorRateEvent(ctx, TopicErrorRateRecovered, stack, open)
	if err != nil {
		return err
	}
	if err := s.errorRates.Close(ctx, open.ID, end, evt); err != nil {
		return fmt.Errorf("close error-rate episode %s: %w", open.ID, err)
	}
	return nil
}

// errorRateEvent builds the outbox row announcing e, written in the same transaction as the episode itself.
func (s *Service) errorRateEvent(ctx context.Context, topic string, stack *Stack, e *ErrorRateEpisode) (eventbus.OutboxEvent, error) {
	var workspaceID string
	if stack.ProjectID != "" && s.projects != nil {
		id, err := s.projects.WorkspaceOf(ctx, stack.ProjectID)
		if err != nil {
			return eventbus.OutboxEvent{}, fmt.Errorf("find workspace of project %s: %w", stack.ProjectID, err)
		}
		workspaceID = id
	}
	return eventbus.OutboxEvent{ID: ids.New(), Topic: topic, Payload: ErrorRateEvent{
		WorkspaceID: workspaceID, ProjectID: stack.ProjectID, StackID: stack.ID, StackName: stack.Name, EpisodeID: e.ID,
		PeakPercent: e.PeakPercent, Requests: e.Requests, Errors: e.Errors, LimitPercent: e.LimitPercent,
		WindowMinutes: e.WindowMinutes, MinRequests: e.MinRequests, StartedAt: e.StartedAt, EndedAt: e.EndedAt,
	}}, nil
}

// errorRateUnavailable says why the rule cannot watch a stack whose routes these are: it has no hostname, or a
// hostname Cloudflare counts and the token cannot read analytics for. "" when it can.
func (s *Service) errorRateUnavailable(ctx context.Context, routes []RequestRoute) string {
	if len(routes) == 0 {
		return ReasonNoHostname
	}
	hosts := map[string][]string{}
	for _, r := range routes {
		if r.Tunnel {
			hosts[r.ZoneID] = append(hosts[r.ZoneID], r.Hostname)
		}
	}
	for zone, names := range hosts {
		if s.analyticsDenied(ctx, zone, names) {
			return ReasonNoAnalytics
		}
	}
	return ""
}

// analyticsDenied asks Cloudflare for a minute of the zone's counts and reports whether it refused the token; any other
// failure is not a refusal. The answer is reused for a minute, since stack reads ask on every refresh.
func (s *Service) analyticsDenied(ctx context.Context, zone string, hostnames []string) bool {
	st := s.requests
	now := s.now()
	st.probeMu.Lock()
	defer st.probeMu.Unlock()
	if p, ok := st.probes[zone]; ok && now.Sub(p.at) < analyticsProbeTTL {
		return p.denied
	}
	end := now.UTC().Add(-requestLag).Truncate(time.Minute)
	_, err := st.sources.TunnelCounts(ctx, zone, hostnames, end.Add(-time.Minute), end)
	denied := errors.Is(err, apperrs.ErrInvalid)
	if st.probes == nil {
		st.probes = map[string]analyticsProbe{}
	}
	st.probes[zone] = analyticsProbe{at: now, denied: denied}
	return denied
}

// requireErrorRateAvailable refuses to switch the rule on for a stack it cannot watch.
func (s *Service) requireErrorRateAvailable(ctx context.Context, before, after *Stack) error {
	if s.requests == nil || !after.ErrorRate.Enabled || before.ErrorRate.Enabled {
		return nil
	}
	routes, err := s.stackRoutes(ctx, after.ID)
	if err != nil {
		return err
	}
	if reason := s.errorRateUnavailable(ctx, routes); reason != "" {
		return fmt.Errorf("%w: the error-rate rule can't be switched on: %s", apperrs.ErrInvalid, reason)
	}
	return nil
}

// percentText reads a share like "7.2%" or "5%".
func percentText(p float64) string {
	return strconv.FormatFloat(math.Round(p*10)/10, 'f', -1, 64) + "%"
}
