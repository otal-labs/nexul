package storage

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/auditlog"
	"github.com/otal-labs/nexul/ee/license"
	"github.com/otal-labs/nexul/internal/integrations"
)

func auditRow(id string, at time.Time) integrations.AuditEntry {
	return integrations.AuditEntry{ID: id, ActorType: "user", ActorID: "u-alice", Action: "PATCH /api/tickets/t1", CreatedAt: at}
}

func dueAuditIDs(t *testing.T, s *Store, now time.Time) []string {
	t.Helper()
	due, err := s.AuditStream.Due(t.Context(), 50, now)
	require.NoError(t, err)
	ids := []string{}
	for _, d := range due {
		ids = append(ids, d.ID)
	}
	return ids
}

func TestAuditRepo_Append_QueuesTheRowInItsOwnWriteOnlyWhileStreaming(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	at := time.Unix(1_000, 0).UTC()

	require.NoError(t, s.Audit.Append(ctx, auditRow("before-wiring", at)), "a store nobody told to stream queues nothing")
	streaming := false
	s.SetAuditStreaming(func(context.Context) bool { return streaming })
	require.NoError(t, s.Audit.Append(ctx, auditRow("off", at)))
	streaming = true
	require.NoError(t, s.Audit.Append(ctx, auditRow("b", at.Add(time.Second))))
	require.NoError(t, s.Audit.Append(ctx, auditRow("a", at)))
	computer := auditRow("on-a-computer", at)
	computer.ComputerID = "computer-1"
	require.NoError(t, s.Audit.Append(ctx, computer), "computer activity is not in the audit log, so it is not streamed")

	assert.Equal(t, []string{"a", "b"}, dueAuditIDs(t, s, at.Add(time.Minute)), "due rows come oldest first")
	assert.Empty(t, dueAuditIDs(t, s, at.Add(-time.Second)), "nothing is due before it was written")
	due, err := s.AuditStream.Due(ctx, 1, at.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, []auditlog.Pending{{ID: "a", ActorType: "user", ActorID: "u-alice", Action: "PATCH /api/tickets/t1", CreatedAt: at}}, due)
}

func TestAuditStreamRepo_MarksCountsAndRetries(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	s.SetAuditStreaming(func(context.Context) bool { return true })
	at := time.Unix(1_000, 0).UTC()
	for _, id := range []string{"sent", "failing", "dead"} {
		require.NoError(t, s.Audit.Append(ctx, auditRow(id, at)))
	}

	st, err := s.AuditStream.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, auditlog.Stats{Pending: 3}, st)

	require.NoError(t, s.AuditStream.MarkDelivered(ctx, "sent", at.Add(time.Minute)))
	require.NoError(t, s.AuditStream.MarkFailed(ctx, "failing", 2, at.Add(time.Hour), "the destination answered 500"))
	require.NoError(t, s.AuditStream.MarkDead(ctx, "dead", 8, "the destination answered 404"))

	st, err = s.AuditStream.Stats(ctx)
	require.NoError(t, err)
	sent := at.Add(time.Minute)
	assert.Equal(t, auditlog.Stats{Pending: 1, Dead: 1, LastDeliveredAt: &sent, LastError: "the destination answered 500"}, st)
	assert.Empty(t, dueAuditIDs(t, s, at.Add(time.Minute)), "a failed row waits for its retry, a dead one for an Owner")
	assert.Equal(t, []string{"failing"}, dueAuditIDs(t, s, at.Add(time.Hour)))

	n, err := s.AuditStream.RetryDead(ctx, at.Add(2*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	due, err := s.AuditStream.Due(ctx, 50, at.Add(2*time.Minute))
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, auditlog.Pending{ID: "dead", ActorType: "user", ActorID: "u-alice", Action: "PATCH /api/tickets/t1", CreatedAt: at}, due[0],
		"a retried row starts its attempts again")
}

func TestAuditStreamRepo_RetentionRemovesTheDeliveriesWithTheirRows(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	s.SetAuditStreaming(func(context.Context) bool { return true })
	old, recent := time.Unix(1_000, 0).UTC(), time.Unix(9_000, 0).UTC()
	require.NoError(t, s.Audit.Append(ctx, auditRow("old", old)))
	require.NoError(t, s.Audit.Append(ctx, auditRow("recent", recent)))

	n, err := s.Audit.DeleteBefore(ctx, recent, 500)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	assert.Equal(t, []string{"recent"}, dueAuditIDs(t, s, recent))
	st, err := s.AuditStream.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, st.Pending)
}

type unlockedAudit struct{}

func (unlockedAudit) Enabled(_ context.Context, f license.Feature) bool {
	return f == license.FeatureAudit
}

// stalled is a destination that never answers, until the request gives up.
type stalled struct{}

func (st *stalled) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

func TestAuditRepo_Append_ABurstFinishesWhileTheDestinationHangs(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	svc := auditlog.NewService(auditlog.Config{
		Repo: s.AuditSettings, Queue: s.AuditStream, Licenses: unlockedAudit{},
		InstanceURL: func(context.Context) (string, error) { return "https://nexul.example.com", nil },
		Client:      &http.Client{Transport: &stalled{}},
	})
	require.NoError(t, s.AuditSettings.Set(ctx, auditlog.Settings{StreamURL: "https://logs.example.com", StreamSecret: "s3cret", StreamEnabled: true, UpdatedAt: time.Now()}))
	s.SetAuditStreaming(svc.Streaming)
	require.NoError(t, s.Audit.Append(ctx, auditRow("first", time.Now())))
	relayCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		svc.NewRelay(s.CommitsTo(AuditStreamWrites...).Next, slog.New(slog.DiscardHandler)).Run(relayCtx)
		close(done)
	}()
	t.Cleanup(func() {
		stop()
		<-done
	})

	for i := range 200 {
		require.NoError(t, s.Audit.Append(ctx, auditRow("burst-"+itoa(int64(i)), time.Now())))
	}

	st, err := s.AuditStream.Stats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 201, st.Pending, "every row is queued while its delivery hangs")
}
