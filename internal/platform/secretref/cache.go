package secretref

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Source is the managers a Cache reads through.
type Source interface {
	Registry() Registry
	States() []ManagerState
}

const (
	// readEvery is the shortest gap between reads of a failing reference once it backed off fully: 100 reads a day.
	readEvery   = 24 * time.Hour / 100
	retryFloor  = 30 * time.Second
	readTimeout = 15 * time.Second
	// reresolveEvery is how often a provider's rejection may send a reference back to its manager.
	reresolveEvery = 5 * time.Minute
)

// Cache keeps resolved values in memory, keyed by the reference that produced them, so a value is read from its
// manager once and never stored anywhere else (ADR 0181). A reference that fails to resolve is not read again until
// its backoff ends.
type Cache struct {
	src Source
	now func() time.Time

	// mu is held across a read: reads happen once per reference and again only after a backoff or a rejection.
	mu      sync.Mutex
	entries map[string]*entry
}

type entry struct {
	value string
	// prev is the value before the last re-resolve, so a second caller holding it is handed the new one.
	prev string
	err  error
	// retryAt is when a failed reference may be read again.
	retryAt      time.Time
	fails        int
	reresolvedAt time.Time
}

// NewCache reads through src; now is time.Now when nil.
func NewCache(src Source, now func() time.Time) *Cache {
	if now == nil {
		now = time.Now
	}
	return &Cache{src: src, now: now, entries: map[string]*entry{}}
}

// Value is stored itself when it is not a reference, else the reference's value: from memory, or read now unless
// an earlier read failed and its backoff is still running, which answers with that failure.
func (c *Cache) Value(ctx context.Context, stored string) (string, error) {
	if kind, _ := Parse(stored); kind != Reference {
		return stored, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entry(stored)
	if e.value != "" {
		return e.value, nil
	}
	if c.now().Before(e.retryAt) {
		return "", e.err
	}
	return c.read(ctx, stored, e)
}

// Read is Value for a save: it always asks the manager, so a bad reference is refused with the manager's answer.
func (c *Cache) Read(ctx context.Context, stored string) (string, error) {
	if kind, _ := Parse(stored); kind != Reference {
		return stored, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.read(ctx, stored, c.entry(stored))
}

// Refresh answers a provider that rejected stale: it reads the reference behind stale again and returns the new
// value so the caller can retry once. It reports false when stale is no reference's value, the reference was read
// again less than 5 minutes ago, the read failed, or the value did not change.
func (c *Cache) Refresh(ctx context.Context, stale string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for stored, e := range c.entries {
		if e.prev == stale && e.value != "" && e.value != stale {
			return e.value, true
		}
		if e.value != stale || stale == "" {
			continue
		}
		if !e.reresolvedAt.IsZero() && c.now().Sub(e.reresolvedAt) < reresolveEvery {
			return "", false
		}
		e.reresolvedAt = c.now()
		fresh, err := c.read(ctx, stored, e)
		if err != nil || fresh == stale {
			return "", false
		}
		e.prev = stale
		return fresh, true
	}
	return "", false
}

func (c *Cache) entry(stored string) *entry {
	e, ok := c.entries[stored]
	if !ok {
		e = &entry{}
		c.entries[stored] = e
	}
	return e
}

// read asks the reference's manager once and records the outcome; a failure keeps the value already held.
func (c *Cache) read(ctx context.Context, stored string, e *entry) (string, error) {
	res, err := c.src.Registry().For(stored)
	if err != nil {
		e.err = c.noResolver(stored)
		return "", e.err
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	results, err := res.ResolveAll(readCtx, []string{stored})
	if err == nil && len(results) != 1 {
		err = ErrFailed
	}
	if err == nil {
		err = results[0].Err
	}
	if err != nil {
		e.fails++
		e.retryAt = c.now().Add(min(retryFloor<<min(e.fails-1, 6), readEvery))
		if e.value == "" {
			e.err = err
		}
		return "", err
	}
	e.fails, e.retryAt, e.err = 0, time.Time{}, nil
	e.value = results[0].Value
	return e.value, nil
}

// noResolver says why no resolver serves stored's scheme: the manager is out of reach, its token was refused, or
// none is set up. It costs no read, so it is never backed off.
func (c *Cache) noResolver(stored string) error {
	_, ref := Parse(stored)
	for _, s := range c.src.States() {
		if s.Manager != ref.Scheme {
			continue
		}
		switch s.State {
		case ManagerUnreachable:
			return fmt.Errorf("%w: %s", ErrUnreachable, ref.Scheme)
		case ManagerTokenRejected:
			return fmt.Errorf("%w: %s", ErrTokenRejected, ref.Scheme)
		}
	}
	return fmt.Errorf("%w %q", ErrNoResolver, ref.Scheme)
}
