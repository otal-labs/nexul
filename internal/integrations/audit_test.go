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

	deleted, err := svc.PurgeAudit(t.Context(), nil)
	require.NoError(t, err)

	assert.Equal(t, int64(2*auditPurgeBatch+3), deleted)
	assert.Len(t, f.audit, 4, "rows newer than 45 days stay")
	assert.Equal(t, []int{auditPurgeBatch, auditPurgeBatch, auditPurgeBatch}, f.purgeCalls(),
		"each write deletes at most one batch, and a short batch ends the pass")
}

func TestPurgeAudit_StoreFails_ReturnsTheError(t *testing.T) {
	f := newFakeData()
	f.purgeErr = errors.New("database is locked")
	_, err := newTestServiceWithOwner(f, true).PurgeAudit(t.Context(), nil)
	require.ErrorIs(t, err, f.purgeErr)
}

func TestPurgeAudit_DaysCannotBeRead_DeletesNothing(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	seedAudit(f, 3, svc.cfg.Now().Add(-400*24*time.Hour))
	readErr := errors.New("database is locked")

	_, err := svc.PurgeAudit(t.Context(), func(context.Context) (int, error) { return 0, readErr })

	require.ErrorIs(t, err, readErr)
	assert.Len(t, f.audit, 3, "a setting that cannot be read never falls back to deleting at 45 days")
	assert.Empty(t, f.purgeCalls())
}

func TestPurgeAudit_KeepsTheSavedDaysInBoundedBatches(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	now := svc.cfg.Now()
	year := 365 * 24 * time.Hour
	seedAudit(f, auditPurgeBatch+2, now.Add(-year-time.Minute))
	seedAudit(f, 5, now.Add(-year+time.Minute))
	seedAudit(f, 7, now.Add(-100*24*time.Hour))

	deleted, err := svc.PurgeAudit(t.Context(), func(context.Context) (int, error) { return 365, nil })
	require.NoError(t, err)

	assert.Equal(t, int64(auditPurgeBatch+2), deleted)
	assert.Len(t, f.audit, 12, "rows inside the saved year stay, those past 45 days included")
	assert.Equal(t, []int{auditPurgeBatch, auditPurgeBatch}, f.purgeCalls())
}

func TestPurgeAudit_DaysBelowTheFloor_Keep45(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	now := svc.cfg.Now()
	seedAudit(f, 2, now.Add(-AuditRetention-time.Minute))
	seedAudit(f, 3, now.Add(-AuditRetention+time.Minute))

	_, err := svc.PurgeAudit(t.Context(), func(context.Context) (int, error) { return 10, nil })
	require.NoError(t, err)

	assert.Len(t, f.audit, 3)
}

func TestPurgeAudit_SettingCleared_TheNextPurgeDeletesTheOlderRows(t *testing.T) {
	f := newFakeData()
	svc := newTestServiceWithOwner(f, true)
	now := svc.cfg.Now()
	seedAudit(f, 4, now.Add(-200*24*time.Hour))
	seedAudit(f, 1, now.Add(-time.Hour))
	days := 365
	keep := func(context.Context) (int, error) { return days, nil }

	_, err := svc.PurgeAudit(t.Context(), keep)
	require.NoError(t, err)
	require.Len(t, f.audit, 5)

	days = 45
	_, err = svc.PurgeAudit(t.Context(), keep)
	require.NoError(t, err)
	assert.Len(t, f.audit, 1)
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
			svc.RunAuditRetention(ctx, nil, slog.New(slog.DiscardHandler))
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

type fakeRetention struct{ cutoffs map[string]time.Time }

func (f *fakeRetention) record(name string, before time.Time) (int64, error) {
	f.cutoffs[name] = before
	return 0, nil
}

func (f *fakeRetention) DeleteProcessedBefore(_ context.Context, before time.Time, _ int) (int64, error) {
	return f.record("processed", before)
}

func (f *fakeRetention) DeleteOutboxBefore(_ context.Context, before time.Time, _ int) (int64, error) {
	return f.record("outbox", before)
}

func (f *fakeRetention) DeleteDeployLogBefore(_ context.Context, before time.Time, _ int) (int64, error) {
	return f.record("deploylog", before)
}

func TestPurgeEvents_PurgesEveryEventTableAtTheAuditCutoff(t *testing.T) {
	retention := &fakeRetention{cutoffs: map[string]time.Time{}}
	svc := newTestServiceWithOwner(newFakeData(), true)
	svc.cfg.Retention = retention

	_, err := svc.PurgeEvents(t.Context())
	require.NoError(t, err)

	want := svc.cfg.Now().Add(-AuditRetention)
	assert.Equal(t, map[string]time.Time{"processed": want, "outbox": want, "deploylog": want}, retention.cutoffs)
}

func TestRunAuditRetention_ALongerSetting_LeavesTheEventsPurgeAt45Days(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		retention := &fakeRetention{cutoffs: map[string]time.Time{}}
		f := newFakeData()
		svc := newTestServiceWithOwner(f, true)
		svc.cfg.Now = time.Now
		svc.cfg.Retention = retention
		seedAudit(f, 2, time.Now().Add(-100*24*time.Hour))
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			svc.RunAuditRetention(ctx, func(context.Context) (int, error) { return 1095, nil }, slog.New(slog.DiscardHandler))
			close(done)
		}()
		synctest.Wait()
		cancel()
		<-done

		assert.Len(t, f.audit, 2, "the saved days keep the audit rows")
		want := time.Now().Add(-AuditRetention)
		assert.Equal(t, map[string]time.Time{"processed": want, "outbox": want, "deploylog": want}, retention.cutoffs)
	})
}
