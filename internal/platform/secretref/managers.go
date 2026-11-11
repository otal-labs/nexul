package secretref

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"slices"
	"sync"
	"time"
)

// Secrets manager states a process reports (ADR 0181). The values are published, so a second manager adds none.
const (
	ManagerReady         = "ready"
	ManagerTokenRejected = "token rejected"
	ManagerUnreachable   = "unreachable"
)

// ManagerState is one secrets manager's state; Manager is the scheme of the references it resolves, so no manager's
// name is published.
type ManagerState struct {
	Manager string `json:"manager" jsonschema:"The secrets manager, named by the scheme of the references it resolves, such as op."`
	State   string `json:"state" jsonschema:"Whether the runner can resolve this manager's references." enum:"ready,token rejected,unreachable"`
}

// Setup opens one secrets manager's resolver from the credential the process holds for it.
type Setup struct {
	// Scheme names the manager, as the references it resolves do.
	Scheme string
	Open   func(ctx context.Context) (Resolver, error)
}

const (
	openTimeout = 30 * time.Second
	retryBase   = 5 * time.Second
	retryMax    = 5 * time.Minute
)

// Managers holds the resolvers a process opened at start and the state each manager is in. A resolver is opened once
// and kept for the process's life; only an unreachable manager is tried again, a rejected token is final.
type Managers struct {
	log    *slog.Logger
	setups []Setup
	base   time.Duration
	max    time.Duration

	mu         sync.Mutex
	states     map[string]string
	resolvers  []Resolver
	changed    chan struct{}
	onRejected func(ctx context.Context, scheme string)
}

// NewManagers prepares the setups; nothing is opened until Start.
func NewManagers(log *slog.Logger, setups ...Setup) *Managers {
	return &Managers{
		log: log, setups: setups, base: retryBase, max: retryMax,
		states: map[string]string{}, changed: make(chan struct{}),
	}
}

// SetRetry changes the wait between attempts on an unreachable manager; call it before Start.
func (m *Managers) SetRetry(base, max time.Duration) { m.base, m.max = base, max }

// SetOnRejected calls fn once each time a manager's state turns token rejected; call it before Start.
func (m *Managers) SetOnRejected(fn func(ctx context.Context, scheme string)) { m.onRejected = fn }

// Start opens every manager once, waiting for the first outcome of each, then keeps retrying the unreachable ones in
// the background until ctx ends. Its goroutines join wg.
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
func (m *Managers) try(ctx context.Context, s Setup) string {
	openCtx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()
	res, err := s.Open(openCtx)
	state := ManagerReady
	if err != nil {
		state = ManagerTokenRejected
		if errors.Is(err, ErrUnreachable) || openCtx.Err() != nil {
			state = ManagerUnreachable
		}
		m.log.Warn("secrets manager not ready", "manager", s.Scheme, "state", state, "error", err)
	}
	if m.set(s.Scheme, state, res) && state == ManagerTokenRejected && m.onRejected != nil {
		m.onRejected(ctx, s.Scheme)
	}
	return state
}

func (m *Managers) retry(ctx context.Context, s Setup) {
	for attempt := 0; m.wait(ctx, attempt); attempt++ {
		if m.try(ctx, s) != ManagerUnreachable {
			return
		}
	}
}

// wait sleeps the attempt's doubling delay plus jitter; false when ctx ended first.
func (m *Managers) wait(ctx context.Context, attempt int) bool {
	d := m.base
	for i := 0; i < attempt && d < m.max; i++ {
		d *= 2
	}
	d = min(d, m.max)
	if d > 1 {
		d += rand.N(d / 2)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// set records a manager's state and reports whether it changed.
func (m *Managers) set(scheme, state string, res Resolver) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.states[scheme] == state {
		return false
	}
	m.states[scheme] = state
	if state == ManagerReady {
		m.resolvers = append(m.resolvers, res)
	}
	close(m.changed)
	m.changed = make(chan struct{})
	return true
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

// Registry picks among the resolvers that opened, for the code that resolves references.
func (m *Managers) Registry() Registry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return NewRegistry(slices.Clone(m.resolvers)...)
}
