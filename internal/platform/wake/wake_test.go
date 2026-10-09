package wake_test

import (
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"

	"github.com/otal-labs/nexul/internal/platform/wake"
)

func TestBroadcast_Notify_WakesEveryWaiter(t *testing.T) {
	var b wake.Broadcast
	first, second := b.Next(), b.Next()

	b.Notify()

	for _, ch := range []<-chan struct{}{first, second} {
		select {
		case <-ch:
		default:
			t.Fatal("a waiter was not woken")
		}
	}
}

func TestBroadcast_Next_AfterANotifyWaitsForTheNextOne(t *testing.T) {
	var b wake.Broadcast
	b.Notify()

	ch := b.Next()
	select {
	case <-ch:
		t.Fatal("a notify before Next must not wake it")
	default:
	}

	b.Notify()
	<-ch
}

func TestBroadcast_Notify_WithNoWaiterIsANoop(t *testing.T) {
	var b wake.Broadcast
	b.Notify()
	b.Notify()
}

func TestBroadcast_ConcurrentWaitersAllWake(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var b wake.Broadcast
		var woken sync.WaitGroup
		var n int
		var mu sync.Mutex
		for range 5 {
			ch := b.Next()
			woken.Go(func() {
				<-ch
				mu.Lock()
				n++
				mu.Unlock()
			})
		}
		synctest.Wait()
		b.Notify()
		woken.Wait()
		assert.Equal(t, 5, n)
	})
}
