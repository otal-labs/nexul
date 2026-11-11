package runner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// The secret check's interval setting (NEXUL_SECRET_CHECK_INTERVAL): an hour by default, never under five minutes,
// and "off" switches the check off.
const (
	defaultSecretCheckInterval = time.Hour
	minSecretCheckInterval     = 5 * time.Minute
	secretCheckOff             = "off"
)

// ParseSecretCheckInterval reads NEXUL_SECRET_CHECK_INTERVAL; zero means the check is off.
func ParseSecretCheckInterval(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultSecretCheckInterval, nil
	}
	if strings.EqualFold(raw, secretCheckOff) {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("NEXUL_SECRET_CHECK_INTERVAL %q is not a duration like 30m or 1h, or %q", raw, secretCheckOff)
	}
	if d < minSecretCheckInterval {
		return 0, fmt.Errorf("NEXUL_SECRET_CHECK_INTERVAL %q is under the %s minimum: each check reads the secrets manager, which limits reads per hour and per day", raw, minSecretCheckInterval)
	}
	return d, nil
}

// reportedVars is what the check already told the server about one deploy of a stack, so a value that stays changed
// is reported once and a value that changes again is reported again.
type reportedVars struct {
	deployID string
	prints   map[string]string
}

// SecretCheck re-reads, every interval, the references of every stack this runner holds fingerprints for, and keeps
// the variables whose value changed until a connection delivers them. It never keeps or sends a value, a reference
// or a fingerprint beyond this process's memory and the runner's own files.
type SecretCheck struct {
	store    *Fingerprints
	registry func() secretref.Registry
	every    time.Duration
	log      *slog.Logger

	mu       sync.Mutex
	reported map[string]reportedVars
	pending  map[string][]string
	wake     chan struct{}
}

// NewSecretCheck prepares the check; with no record store, no registry or an interval of zero there is nothing to
// check, and it returns nil.
func NewSecretCheck(store *Fingerprints, registry func() secretref.Registry, every time.Duration, log *slog.Logger) *SecretCheck {
	if store == nil || registry == nil || every <= 0 {
		return nil
	}
	return &SecretCheck{
		store: store, registry: registry, every: every, log: log,
		reported: map[string]reportedVars{}, pending: map[string][]string{}, wake: make(chan struct{}, 1),
	}
}

// run checks on every tick until ctx ends; the first check comes one interval after start.
func (s *SecretCheck) run(ctx context.Context) {
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.check(ctx)
		}
	}
}

// check resolves every held reference in one batch per manager, whatever the number of stacks, and queues the
// variables whose fingerprint differs from the one their stack's last deploy recorded.
func (s *SecretCheck) check(ctx context.Context) {
	records := s.records()
	if len(records) == 0 {
		return
	}
	reg := s.registry()
	if len(reg) == 0 {
		return
	}
	values := s.resolve(ctx, reg, records)
	for _, slug := range slices.Sorted(maps.Keys(records)) {
		s.compare(slug, records[slug], values)
	}
	s.forgetGone(records)
}

// records reads every stack's record; one that cannot be read is skipped until its next deploy rewrites it.
func (s *SecretCheck) records() map[string]stackFingerprints {
	slugs, err := s.store.slugs()
	if err != nil {
		s.log.Warn("secret check cannot list its records", "error", err)
		return nil
	}
	records := map[string]stackFingerprints{}
	for _, slug := range slugs {
		rec, err := s.store.load(slug)
		if err != nil {
			s.log.Warn("secret check skips an unreadable record", "slug", slug, "error", err)
			continue
		}
		records[slug] = rec
	}
	return records
}

// resolve reads every distinct reference once, a batch per scheme, keyed by "<slug>/<variable>"; a variable that did
// not resolve is left out and the check goes on with the rest.
func (s *SecretCheck) resolve(ctx context.Context, reg secretref.Registry, records map[string]stackFingerprints) map[string]string {
	env := map[string]string{}
	byScheme := map[string][]string{}
	for slug, rec := range records {
		for name, v := range rec.Vars {
			kind, ref := secretref.Parse(v.Ref)
			if kind != secretref.Reference {
				continue
			}
			key := valueKey(slug, name)
			env[key] = v.Ref
			byScheme[ref.Scheme] = append(byScheme[ref.Scheme], key)
		}
	}
	values := map[string]string{}
	unresolved := 0
	for scheme, keys := range byScheme {
		got, failed := resolveBatch(ctx, reg, scheme, env, keys)
		maps.Copy(values, got)
		unresolved += len(failed)
	}
	if unresolved > 0 {
		s.log.Warn("secret check could not read some variables; they are tried again next time", "count", unresolved)
	}
	return values
}

func valueKey(slug, name string) string { return slug + "/" + name }

func (s *SecretCheck) compare(slug string, rec stackFingerprints, values map[string]string) {
	prints := map[string]string{}
	for name, v := range rec.Vars {
		value, ok := values[valueKey(slug, name)]
		if !ok {
			continue
		}
		fp, err := s.store.print(value)
		if err != nil {
			s.log.Warn("secret check cannot fingerprint", "error", err)
			return
		}
		if fp != v.Print {
			prints[name] = fp
		}
	}
	if len(prints) == 0 {
		return
	}
	s.queue(slug, rec.DeployID, prints)
}

// queue keeps the variables of a stack not yet reported for this deploy. A stack deployed again while the check was
// reading belongs to its new deploy, so it is left for the next check.
func (s *SecretCheck) queue(slug, deployID string, prints map[string]string) {
	cur, err := s.store.load(slug)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && cur.DeployID != deployID) {
		return
	}
	if err != nil {
		s.log.Warn("secret check skips an unreadable record", "slug", slug, "error", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rep := s.reported[slug]
	if rep.deployID != deployID {
		rep = reportedVars{deployID: deployID}
	}
	if rep.prints == nil {
		rep.prints = map[string]string{}
	}
	var fresh []string
	for name, fp := range prints {
		if rep.prints[name] == fp {
			continue
		}
		rep.prints[name] = fp
		fresh = append(fresh, name)
	}
	s.reported[slug] = rep
	if len(fresh) == 0 {
		return
	}
	s.pending[slug] = slices.Compact(slices.Sorted(slices.Values(slices.Concat(s.pending[slug], fresh))))
	s.signal()
}

// forgetGone drops what was reported or waits for delivery for a stack that no longer has a record.
func (s *SecretCheck) forgetGone(records map[string]stackFingerprints) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for slug := range s.reported {
		if _, ok := records[slug]; !ok {
			delete(s.reported, slug)
			delete(s.pending, slug)
		}
	}
}

func (s *SecretCheck) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// take hands over the changes waiting for delivery.
func (s *SecretCheck) take() []SecretChange {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []SecretChange
	for _, slug := range slices.Sorted(maps.Keys(s.pending)) {
		out = append(out, SecretChange{Slug: slug, Variables: s.pending[slug]})
	}
	clear(s.pending)
	return out
}

// restore puts back changes a connection failed to deliver, so the next connection sends them.
func (s *SecretCheck) restore(changes []SecretChange) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range changes {
		s.pending[c.Slug] = slices.Compact(slices.Sorted(slices.Values(slices.Concat(s.pending[c.Slug], c.Variables))))
	}
}

// reportSecrets sends the check's findings on one connection, as soon as there are any, until it ends.
func (c *Client) reportSecrets(ctx context.Context, conn *websocket.Conn) {
	check := c.cfg.SecretCheck
	for {
		if changes := check.take(); len(changes) > 0 {
			if !c.sendSecretChanges(ctx, conn, changes) {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-check.wake:
		}
	}
}

// sendSecretChanges reports false when the connection is gone; what it could not send stays queued. A finding the
// server would refuse is dropped, since a refused frame closes the connection and would be refused again.
func (c *Client) sendSecretChanges(ctx context.Context, conn *websocket.Conn, changes []SecretChange) bool {
	f := Frame{Type: FrameSecretsChanged, SecretChanges: changes}
	if err := f.Validate(); err != nil {
		c.log.Warn("secret check finding dropped", "runner", c.cfg.Name, "error", err)
		return true
	}
	if err := c.writeFrame(ctx, conn, f); err != nil {
		c.cfg.SecretCheck.restore(changes)
		c.log.Warn("frame send failed", "runner", c.cfg.Name, "type", f.Type, "error", err)
		return false
	}
	return true
}
