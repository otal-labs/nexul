package runner

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand"
	"slices"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/platform/secretref"
)

// ManagerSetup opens one secrets manager's resolver from the credential this runner holds for it.
type ManagerSetup struct {
	// Scheme names the manager, as the references it resolves do.
	Scheme string
	Open   func(ctx context.Context) (secretref.Resolver, error)
}

const (
	managerOpenTimeout  = 30 * time.Second
	managerRetryBase    = 5 * time.Second
	managerRetryMax     = 5 * time.Minute
	maxReportedManagers = 8
)

// Managers holds the resolvers a runner opened at start and the state each manager is in. A resolver is opened once
// and kept for the process's life; only an unreachable manager is tried again.
type Managers struct {
	log    *slog.Logger
	setups []ManagerSetup
	base   time.Duration
	max    time.Duration

	mu        sync.Mutex
	states    map[string]string
	resolvers []secretref.Resolver
	changed   chan struct{}
}

// NewManagers prepares the setups; nothing is opened until Start.
func NewManagers(log *slog.Logger, setups ...ManagerSetup) *Managers {
	return &Managers{
		log: log, setups: setups, base: managerRetryBase, max: managerRetryMax,
		states: map[string]string{}, changed: make(chan struct{}),
	}
}

// Start opens every manager once, waiting for the first outcome of each so the first connect reports it, then keeps
// retrying the unreachable ones in the background until ctx ends. Its goroutines join wg.
func (m *Managers) Start(ctx context.Context, wg *sync.WaitGroup) {
	var first sync.WaitGroup
	for _, s := range m.setups {
		first.Go(func() {
			if m.try(ctx, s) != ManagerUnreachable {
				return
			}
			wg.Go(func() { m.retry(ctx, s) })
		})
	}
	first.Wait()
}

// try makes one attempt and records its state; a failure that is not the manager being out of reach is a rejected token.
func (m *Managers) try(ctx context.Context, s ManagerSetup) string {
	openCtx, cancel := context.WithTimeout(ctx, managerOpenTimeout)
	defer cancel()
	res, err := s.Open(openCtx)
	state := ManagerReady
	if err != nil {
		state = ManagerTokenRejected
		if errors.Is(err, secretref.ErrUnreachable) || openCtx.Err() != nil {
			state = ManagerUnreachable
		}
		m.log.Warn("secrets manager not ready", "manager", s.Scheme, "state", state, "error", err)
	}
	m.set(s.Scheme, state, res)
	return state
}

func (m *Managers) retry(ctx context.Context, s ManagerSetup) {
	backoff := NewBackoff(m.base, m.max, rand.New(rand.NewSource(time.Now().UnixNano())))
	for attempt := 0; backoff.Wait(ctx, attempt); attempt++ {
		if m.try(ctx, s) != ManagerUnreachable {
			return
		}
	}
}

func (m *Managers) set(scheme, state string, res secretref.Resolver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.states[scheme] == state {
		return
	}
	m.states[scheme] = state
	if state == ManagerReady {
		m.resolvers = append(m.resolvers, res)
	}
	close(m.changed)
	m.changed = make(chan struct{})
}

// States is each manager's state in setup order; empty when no manager was set up.
func (m *Managers) States() []ManagerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ManagerState, 0, len(m.setups))
	for _, s := range m.setups {
		if state, ok := m.states[s.Scheme]; ok {
			out = append(out, ManagerState{Manager: s.Scheme, State: state})
		}
	}
	return out
}

// Changed is closed when a manager's state next changes.
func (m *Managers) Changed() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.changed
}

// Registry picks among the resolvers that opened, for the deploy that resolves references.
func (m *Managers) Registry() secretref.Registry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return secretref.NewRegistry(slices.Clone(m.resolvers)...)
}

// encodeManagers is the connect query's value; empty when there is nothing to report.
func encodeManagers(states []ManagerState) string {
	if len(states) == 0 {
		return ""
	}
	raw, err := json.Marshal(states)
	if err != nil {
		return ""
	}
	return string(raw)
}

// parseManagers reads a connect query's managers; a malformed report, or an entry that does not name a known state,
// is dropped rather than refusing the connection.
func parseManagers(raw string) []ManagerState {
	if raw == "" {
		return nil
	}
	var in []ManagerState
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return nil
	}
	var out []ManagerState
	for _, s := range in {
		if len(out) == maxReportedManagers {
			break
		}
		known := s.State == ManagerReady || s.State == ManagerTokenRejected || s.State == ManagerUnreachable
		if s.Manager == "" || len(s.Manager) > 32 || !known {
			continue
		}
		out = append(out, s)
	}
	return out
}
