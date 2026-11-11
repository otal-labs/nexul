package secretref

import (
	"context"
	"errors"
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
	// timeout bounds one read of a manager.
	timeout time.Duration

	// mu guards entries and what they hold; it is never held across a read, so one slow reference blocks no other.
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
	// flight is the read in progress; callers of the same reference wait on it instead of reading again.
	flight *flight
}

type flight struct {
	done  chan struct{}
	value string
	err   error
}

// NewCache reads through src; now is time.Now when nil.
func NewCache(src Source, now func() time.Time) *Cache {
	if now == nil {
		now = time.Now
	}
	return &Cache{src: src, now: now, timeout: readTimeout, entries: map[string]*entry{}}
}

// Value is stored itself when it is not a reference, else the reference's value: from memory, or read now unless
// an earlier read failed and its backoff is still running, which answers with that failure.
func (c *Cache) Value(ctx context.Context, stored string) (string, error) {
	if kind, _ := Parse(stored); kind != Reference {
		return stored, nil
	}
	c.mu.Lock()
	e := c.entry(stored)
	value, failure, held := e.value, e.err, e.value != "" || c.now().Before(e.retryAt)
	c.mu.Unlock()
	if held {
		return value, failure
	}
	return c.read(ctx, stored)
}

// Read is Value for a save: it always asks the manager, so a bad reference is refused with the manager's answer.
func (c *Cache) Read(ctx context.Context, stored string) (string, error) {
	if kind, _ := Parse(stored); kind != Reference {
		return stored, nil
	}
	return c.read(ctx, stored)
}

// Refresh answers a provider that rejected stale: it reads the reference behind stale again and returns the new
// value so the caller can retry once. It reports false when stale is no reference's value, the reference was read
// again less than 5 minutes ago, the read failed, or the value did not change.
func (c *Cache) Refresh(ctx context.Context, stale string) (string, bool) {
	if stale == "" {
		return "", false
	}
	c.mu.Lock()
	var stored string
	var held *entry
	for s, e := range c.entries {
		if e.prev == stale && e.value != "" && e.value != stale {
			fresh := e.value
			c.mu.Unlock()
			return fresh, true
		}
		if e.value == stale {
			stored, held = s, e
		}
	}
	if held == nil || (!held.reresolvedAt.IsZero() && c.now().Sub(held.reresolvedAt) < reresolveEvery) {
		c.mu.Unlock()
		return "", false
	}
	held.reresolvedAt = c.now()
	c.mu.Unlock()
	fresh, err := c.read(ctx, stored)
	if err != nil || fresh == stale {
		return "", false
	}
	c.mu.Lock()
	held.prev = stale
	c.mu.Unlock()
	return fresh, true
}

// Warm reads every reference in stored now, one call per manager, and seeds the cache with what came back, so the
// first lookup of each finds it in memory. A reference that fails is recorded like any failed read.
func (c *Cache) Warm(ctx context.Context, stored []string) {
	byScheme := map[string][]string{}
	seen := map[string]bool{}
	for _, v := range stored {
		kind, ref := Parse(v)
		if kind != Reference || seen[v] {
			continue
		}
		seen[v] = true
		byScheme[ref.Scheme] = append(byScheme[ref.Scheme], v)
	}
	for _, refs := range byScheme {
		c.warmManager(ctx, refs)
	}
}

func (c *Cache) warmManager(ctx context.Context, refs []string) {
	res, err := c.src.Registry().For(refs[0])
	if err != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, r := range refs {
			c.entry(r).err = c.noResolver(r)
		}
		return
	}
	results, err := c.resolve(ctx, res, refs)
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, r := range refs {
		if err != nil {
			c.record(c.entry(r), "", err)
			continue
		}
		c.record(c.entry(r), results[i].Value, results[i].Err)
	}
}

func (c *Cache) entry(stored string) *entry {
	e, ok := c.entries[stored]
	if !ok {
		e = &entry{}
		c.entries[stored] = e
	}
	return e
}

// read asks the reference's manager once and records the outcome, outside the lock; callers of the same reference
// that arrive meanwhile share the answer.
func (c *Cache) read(ctx context.Context, stored string) (string, error) {
	c.mu.Lock()
	e := c.entry(stored)
	if f := e.flight; f != nil {
		c.mu.Unlock()
		select {
		case <-f.done:
			return f.value, f.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	f := &flight{done: make(chan struct{})}
	e.flight = f
	c.mu.Unlock()

	res, err := c.src.Registry().For(stored)
	if err != nil {
		f.err = c.noResolver(stored)
	}
	if err == nil {
		f.value, f.err = c.fetch(ctx, res, stored)
	}

	c.mu.Lock()
	e.flight = nil
	if err != nil {
		e.err = f.err
	}
	if err == nil {
		c.record(e, f.value, f.err)
	}
	c.mu.Unlock()
	close(f.done)
	return f.value, f.err
}

func (c *Cache) fetch(ctx context.Context, res Resolver, stored string) (string, error) {
	results, err := c.resolve(ctx, res, []string{stored})
	if err != nil {
		return "", err
	}
	return results[0].Value, results[0].Err
}

// resolve is one ResolveAll bounded by the read timeout; a call that ran out of time reads as unreachable.
func (c *Cache) resolve(ctx context.Context, res Resolver, refs []string) ([]Result, error) {
	readCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	results, err := res.ResolveAll(readCtx, refs)
	if err != nil && errors.Is(readCtx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("%w: read timed out", ErrUnreachable)
	}
	if err == nil && len(results) != len(refs) {
		err = ErrFailed
	}
	return results, err
}

// record stores one read's outcome and the caller holds mu; a failure keeps the value already held.
func (c *Cache) record(e *entry, value string, err error) {
	if err == nil {
		e.fails, e.retryAt, e.err = 0, time.Time{}, nil
		e.value = value
		return
	}
	e.fails++
	e.retryAt = c.now().Add(min(retryFloor<<min(e.fails-1, 6), readEvery))
	if e.value == "" {
		e.err = err
	}
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
