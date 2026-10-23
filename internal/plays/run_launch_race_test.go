package plays_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/plays"
)

// Two starts for one person that look at the computer's room at the same moment must not both take its last slot.
func TestLaunch_TwoStartsAtOnce_OnlyOneTakesTheLastSlot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := openStore(t)
		r := newRigOn(&rig{clock: time.Now().UTC()}, store, time.Now)
		fix := r.play(t, "Fix it")
		r.autoPlay(t, fix.ID, plays.MomentTicketUnblocked)
		for _, id := range []string{"t-1", "t-2"} {
			r.ticket(id)
			r.unblock(t, id)
		}

		secondPass := make(chan error, 1)
		r.OnNextResolve(func() {
			go func() {
				_, err := r.Dispatch(t.Context())
				secondPass <- err
			}()
			// Without the lock the second pass has already counted 0 of 1 and started its run by now.
			synctest.Wait()
		})
		r.dispatch(t)
		synctest.Wait()
		require.NoError(t, <-secondPass)

		running, err := store.Pairing.CountRunning(t.Context(), []string{"c-" + dev})
		require.NoError(t, err)
		assert.Equal(t, 1, running["c-"+dev], "a computer that runs one at once runs one")
		got := append(statuses(r.queued(t, "t-1")), statuses(r.queued(t, "t-2"))...)
		assert.ElementsMatch(t, []plays.QueueStatus{plays.QueueStarted, plays.QueueQueued}, got, "the other run waits for room")
	})
}
