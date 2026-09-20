// Package presence holds one harness connection per paired computer while its owner has a live browser WS.
package presence

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/pairing"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// DefaultLinger keeps connections up briefly after the last browser socket closes, so a page refresh doesn't flap.
const DefaultLinger = 10 * time.Second

const (
	initialBackoff = time.Second
	maxBackoff     = time.Minute
)

// Config wires a Keeper.
type Config struct {
	// Sessions lists userID's unexpired computers with decrypted bearer tokens; wired to pairing.Service.ActiveSessions.
	Sessions func(ctx context.Context, userID string) ([]pairing.Computer, error)
	// Harnesses holds the client per kind; each computer is held through its own kind's Hold.
	Harnesses harness.Registry
	Logger    *slog.Logger
	// Linger overrides DefaultLinger (tests use a tiny value).
	Linger time.Duration
}

// Keeper tracks per-user browser WS refcounts and reconciles held harness connections against the user's paired computers.
type Keeper struct {
	cfg Config
	log *slog.Logger

	mu    sync.Mutex
	users map[string]*presence
}

type presence struct {
	refs   int
	ctx    context.Context
	cancel context.CancelFunc
	// computers maps computer ID -> maintain loop; a loop that exits removes its own entry so Refresh can restart it.
	computers map[string]*loop
	linger    *time.Timer
}

// loop is one maintain goroutine's identity, compared by pointer so a stale cleanup can't remove its replacement.
type loop struct {
	cancel context.CancelFunc
	state  string
}

// Presence states surfaced per computer (Status); no entry means nothing is held.
const (
	StateConnected  = "connected"
	StateConnecting = "connecting"
)

// Status reports userID's per-computer presence for the settings row's dot; a missing entry holds no session.
func (k *Keeper) Status(userID string) map[string]string {
	k.mu.Lock()
	defer k.mu.Unlock()
	p, ok := k.users[userID]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(p.computers))
	for id, l := range p.computers {
		out[id] = l.state
	}
	return out
}

func (k *Keeper) setState(l *loop, state string) {
	k.mu.Lock()
	l.state = state
	k.mu.Unlock()
}

// New wires a Keeper. A nil logger defaults to slog.Default().
func New(cfg Config) *Keeper {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Linger <= 0 {
		cfg.Linger = DefaultLinger
	}
	return &Keeper{cfg: cfg, log: cfg.Logger, users: make(map[string]*presence)}
}

// Connected records one more live browser socket for userID; the first one dials their paired computers.
func (k *Keeper) Connected(userID string) {
	if userID == "" {
		return
	}
	k.mu.Lock()
	p := k.users[userID]
	if p == nil {
		ctx, cancel := context.WithCancel(context.Background())
		p = &presence{ctx: ctx, cancel: cancel, computers: make(map[string]*loop)}
		k.users[userID] = p
	}
	p.refs++
	if p.linger != nil {
		p.linger.Stop()
		p.linger = nil
	}
	k.mu.Unlock()
	go k.reconcile(userID)
}

// Disconnected records one browser socket closing; the last one tears down held connections after the linger window.
func (k *Keeper) Disconnected(userID string) {
	if userID == "" {
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	p := k.users[userID]
	if p == nil {
		return
	}
	p.refs--
	if p.refs > 0 {
		return
	}
	if p.linger != nil {
		p.linger.Stop()
	}
	p.linger = time.AfterFunc(k.cfg.Linger, func() {
		k.mu.Lock()
		defer k.mu.Unlock()
		if cur := k.users[userID]; cur == p && p.refs <= 0 {
			p.cancel()
			delete(k.users, userID)
		}
	})
}

// Refresh reconciles held connections against current paired computers; a no-op with no live browser socket.
func (k *Keeper) Refresh(userID string) {
	go k.reconcile(userID)
}

func (k *Keeper) reconcile(userID string) {
	k.mu.Lock()
	p := k.users[userID]
	if p == nil || p.refs <= 0 {
		k.mu.Unlock()
		return
	}
	ctx := p.ctx
	k.mu.Unlock()

	computers, err := k.cfg.Sessions(ctx, userID)
	if err != nil {
		k.log.Warn("presence: list sessions failed", "user", userID, "error", err)
		return
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if k.users[userID] != p {
		return // torn down while we were reading
	}
	desired := make(map[string]pairing.Computer, len(computers))
	for _, c := range computers {
		desired[c.ID] = c
	}
	for id, l := range p.computers {
		if _, ok := desired[id]; !ok {
			l.cancel()
			delete(p.computers, id)
		}
	}
	for id, computer := range desired {
		if _, ok := p.computers[id]; ok {
			continue
		}
		ctx, cancel := context.WithCancel(p.ctx)
		l := &loop{cancel: cancel, state: StateConnecting}
		p.computers[id] = l
		go k.maintain(ctx, userID, p, l, computer)
	}
}

// maintain holds one connection open, redialing with backoff, and removes itself from p.computers on return.
func (k *Keeper) maintain(ctx context.Context, userID string, p *presence, self *loop, computer pairing.Computer) {
	defer func() {
		k.mu.Lock()
		if p.computers[computer.ID] == self {
			delete(p.computers, computer.ID)
		}
		k.mu.Unlock()
	}()
	client, ok := k.cfg.Harnesses[computer.Kind]
	if !ok {
		k.log.Warn("presence: no harness client for kind", "user", userID, "computer", computer.Name, "kind", computer.Kind)
		return
	}
	backoff := initialBackoff
	for {
		if !computer.TokenExpiresAt.After(time.Now()) {
			return
		}
		k.setState(self, StateConnecting)
		conn, err := client.Hold(ctx, computer.Session())
		if err != nil {
			// Re-pairing fires OnComputersChanged -> Refresh; redialing a rejected session can't fix it.
			if errors.Is(err, apperrs.ErrUnauthorized) {
				k.log.Warn("presence: harness rejected session, giving up until re-pair", "user", userID, "computer", computer.Name)
				return
			}
			// Transient failures retry, but silently invisible ones cost a debugging session.
			k.log.Warn("presence: dial failed, retrying", "user", userID, "computer", computer.Name, "backoff", backoff, "error", err)
		}
		if err == nil {
			k.log.Debug("presence: connected", "user", userID, "computer", computer.Name)
			k.setState(self, StateConnected)
			backoff = initialBackoff
			if conn == nil {
				// Nothing to hold for this harness: stay connected until the user's presence is torn down.
				<-ctx.Done()
				return
			}
			select {
			case <-ctx.Done():
				_ = conn.Close()
				return
			case <-conn.Done():
				_ = conn.Close()
				k.setState(self, StateConnecting)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
