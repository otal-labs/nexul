package issuesync

import (
	"sync"
	"time"
)

// GitHub asks for a second between writes and limits content creation to about 80 a minute and 500 an hour, per
// installation; an installation is an owner's, so each owner gets one write a second and createsPerHour creations.
const (
	writeGap       = time.Second
	createsPerHour = 400
)

// pacer spaces each owner's GitHub calls and holds them while the owner's rate limit rests; the sweep and the push
// loop share it from their own goroutines.
type pacer struct {
	mu      sync.Mutex
	resting map[string]time.Time
	wrote   map[string]time.Time
	created map[string][]time.Time
}

func newPacer() *pacer {
	return &pacer{resting: map[string]time.Time{}, wrote: map[string]time.Time{}, created: map[string][]time.Time{}}
}

// restingUntil is when owner's rate limit resets, zero when it is not resting at now.
func (p *pacer) restingUntil(owner string, now time.Time) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	until, ok := p.resting[owner]
	if ok && now.Before(until) {
		return until
	}
	delete(p.resting, owner)
	return time.Time{}
}

func (p *pacer) rest(owner string, until time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resting[owner] = until
}

// wait is how long owner's next write, a creation when create, must wait from now.
func (p *pacer) wait(owner string, now time.Time, create bool) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	ready := p.resting[owner]
	if next := p.wrote[owner].Add(writeGap); next.After(ready) {
		ready = next
	}
	if created := p.recent(owner, now); create && len(created) >= createsPerHour {
		if next := created[len(created)-createsPerHour].Add(time.Hour); next.After(ready) {
			ready = next
		}
	}
	return max(ready.Sub(now), 0)
}

// took records a write owner sent at now.
func (p *pacer) took(owner string, now time.Time, create bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wrote[owner] = now
	if create {
		p.created[owner] = append(p.recent(owner, now), now)
	}
}

// recent is owner's creations within the last hour; callers hold mu.
func (p *pacer) recent(owner string, now time.Time) []time.Time {
	created := p.created[owner]
	i := 0
	for i < len(created) && !created[i].After(now.Add(-time.Hour)) {
		i++
	}
	p.created[owner] = created[i:]
	return p.created[owner]
}
