package botwebhook

import (
	"sync"
	"time"
)

// limiter counts hits per key over a sliding window.
// ponytail: in memory in one process, so a restart forgets every count; move the counts to SQLite if Nexul ever runs more than one server.
type limiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	hits      map[string][]time.Time
	lastSweep time.Time
	now       func() time.Time
}

func newLimiter(limit int, window time.Duration) *limiter {
	return &limiter{limit: limit, window: window, hits: map[string][]time.Time{}, now: time.Now}
}

// quota is a key's place in its window: what is left and how long until the oldest hit frees a slot.
type quota struct {
	limit, remaining int
	resetAfter       time.Duration
}

// peek reports key's quota without spending any.
func (l *limiter) peek(key string) quota {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.quota(key, l.now())
}

// take spends one hit of key's quota if any is left, reporting the quota after it.
func (l *limiter) take(key string) (quota, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	if q := l.quota(key, now); q.remaining == 0 {
		return q, false
	}
	l.hits[key] = append(l.hits[key], now)
	return l.quota(key, now), true
}

// give returns key's latest hit, for a post that took a slot and then failed.
func (l *limiter) give(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n := len(l.hits[key]); n > 0 {
		l.hits[key] = l.hits[key][:n-1]
	}
}

func (l *limiter) quota(key string, now time.Time) quota {
	kept := l.recent(l.hits[key], now)
	if len(kept) < len(l.hits[key]) {
		l.hits[key] = kept
	}
	q := quota{limit: l.limit, remaining: max(l.limit-len(kept), 0)}
	if len(kept) > 0 {
		q.resetAfter = kept[0].Add(l.window).Sub(now)
	}
	return q
}

// sweep forgets keys with no hit left in the window, at most once a window.
func (l *limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	l.lastSweep = now
	for key, times := range l.hits {
		if len(l.recent(times, now)) == 0 {
			delete(l.hits, key)
		}
	}
}

func (l *limiter) recent(times []time.Time, now time.Time) []time.Time {
	i := 0
	for i < len(times) && now.Sub(times[i]) >= l.window {
		i++
	}
	return times[i:]
}
