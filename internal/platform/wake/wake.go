// Package wake is a broadcast that any number of loops can wait on, so one event reaches all of them.
package wake

import "sync"

// Broadcast wakes every waiter at each Notify. The zero value is ready to use and costs nothing until someone waits.
type Broadcast struct {
	mu sync.Mutex
	ch chan struct{}
}

// Next returns a channel that Notify closes. Take it before reading the data a notify announces, so a notify that
// lands during the read still wakes the next wait instead of being lost.
func (b *Broadcast) Next() <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ch == nil {
		b.ch = make(chan struct{})
	}
	return b.ch
}

// Notify wakes everyone holding a channel from Next; later calls to Next get a fresh one.
func (b *Broadcast) Notify() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ch == nil {
		return
	}
	close(b.ch)
	b.ch = nil
}
