package integrations

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedAudit(f *fakeData, n int, at time.Time) {
	for range n {
		f.audit = append(f.audit, AuditEntry{ID: itoa(len(f.audit)), ActorType: "user", ActorID: "u-alice", Action: "PATCH /api/tickets/t1", CreatedAt: at})
	}
}

func TestPurgeAudit_DeletesRowsOlderThan45DaysInBoundedBatches(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	now := svc.cfg.Now()
	fortyFiveDays := 45 * 24 * time.Hour
	seedAudit(f, 2*auditPurgeBatch+3, now.Add(-fortyFiveDays-time.Minute))
	seedAudit(f, 4, now.Add(-fortyFiveDays+time.Minute))

	deleted, err := svc.PurgeAudit(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int64(2*auditPurgeBatch+3), deleted)
	assert.Len(t, f.audit, 4, "rows newer than 45 days stay")
	assert.Equal(t, []int{auditPurgeBatch, auditPurgeBatch, auditPurgeBatch}, f.purgeCalls(),
		"each write deletes at most one batch, and a short batch ends the pass")
}

func TestPurgeAudit_StoreFails_ReturnsTheError(t *testing.T) {
	f := newFakeData()
	f.purgeErr = errors.New("database is locked")
	_, err := newTestServiceWithOwner(f, true).PurgeAudit(t.Context())
	require.ErrorIs(t, err, f.purgeErr)
}

func TestRunAuditRetention_PurgesAtStartThenDaily_SurvivingAFailedPass(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFakeData()
		f.purgeErr = errors.New("database is locked")
		svc := newTestServiceWithOwner(f, true)
		svc.cfg.Now = time.Now
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			svc.RunAuditRetention(ctx, slog.New(slog.DiscardHandler))
			close(done)
		}()

		synctest.Wait()
		assert.Len(t, f.purgeCalls(), 1, "the first pass runs at start")

		time.Sleep(auditRetentionInterval - time.Second)
		synctest.Wait()
		assert.Len(t, f.purgeCalls(), 1)
		time.Sleep(time.Second)
		synctest.Wait()
		assert.Len(t, f.purgeCalls(), 2, "a failed pass does not stop the next day's")

		cancel()
		<-done
	})
}
