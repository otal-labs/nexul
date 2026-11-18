package issuesync

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/wake"
)

func swept(f *fixture, at time.Time) {
	s := f.repo.syncs["acme/app"]
	s.LastSweptAt = &at
	f.repo.syncs["acme/app"] = s
}

func TestSweep_Backfill_LinksTheOpenLabelledIssuesOnce(t *testing.T) {
	f := newFixture(onSync())
	f.issues.pages[1] = []Issue{newIssue()}
	f.issues.next[1] = 2
	f.issues.pages[2] = []Issue{newIssue(func(i *Issue) { i.Number, i.ID = 8, 9002 })}

	require.NoError(t, f.svc.Sweep(t.Context(), false))

	assert.Equal(t, []IssueQuery{{State: "open", Labels: []string{"nexul"}, Page: 1}, {State: "open", Labels: []string{"nexul"}, Page: 2}}, f.issues.queries)
	assert.Len(t, f.desk.tickets, 2)
	assert.Equal(t, f.now, *f.repo.syncs["acme/app"].LastSweptAt)

	handle(t, f, "opened", newIssue())
	require.NoError(t, f.svc.Sweep(t.Context(), false))
	assert.Len(t, f.desk.tickets, 2, "neither the webhook nor a wake after the backfill files the issue again")
	assert.Len(t, f.issues.queries, 2, "a wake sweeps only syncs not yet backfilled")
}

func TestSweep_Since_CatchesWhatWebhooksMissed(t *testing.T) {
	f := newFixture(onSync())
	handle(t, f, "opened", newIssue())
	last := issueTime.Add(-time.Hour)
	swept(f, last)
	f.issues.pages[1] = []Issue{newIssue(withLabels("bug"), later(time.Minute)), newIssue(func(i *Issue) { i.Number, i.Title = 9, "New" })}
	f.issues.comments[7] = []Comment{{ID: 41, Body: "Missed comment", Author: "carol-gh"}}

	require.NoError(t, f.svc.Sweep(t.Context(), true))

	assert.Equal(t, []IssueQuery{{State: "all", Since: last, Page: 1}}, f.issues.queries, "every changed issue, labelled or not, so a removed label is seen")
	assert.True(t, f.repo.links[linkKey{"acme", "app", 7}].Paused, "the label removal the webhook missed pauses the issue")
	assert.Len(t, f.desk.tickets, 2, "an issue labelled while its webhook was lost is filed")
	assert.Empty(t, f.desk.messages, "a paused issue's comments wait")

	handle(t, f, "labeled", newIssue(later(2*time.Minute)))
	f.issues.pages[1] = []Issue{newIssue(later(2 * time.Minute))}
	swept(f, last)
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Equal(t, map[string]string{"m-41": "Missed comment"}, f.desk.messages)
}

func TestSweep_RateLimitLow_RestsUntilTheReset(t *testing.T) {
	f := newFixture(onSync())
	f.issues.rate = Rate{Remaining: 199, Reset: f.now.Add(10 * time.Minute)}
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Empty(t, f.issues.queries)
	assert.Nil(t, f.repo.syncs["acme/app"].LastSweptAt, "a paused sweep is not recorded, so it resumes from the same point")

	f.issues.rate = Rate{Remaining: 5000}
	f.now = f.now.Add(5 * time.Minute)
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Empty(t, f.issues.queries, "it still waits for the reset")

	f.now = f.now.Add(6 * time.Minute)
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Len(t, f.issues.queries, 1)
	assert.NotNil(t, f.repo.syncs["acme/app"].LastSweptAt)
}

func TestSweep_Refused_RestsForTheWaitGitHubGave(t *testing.T) {
	f := newFixture(onSync())
	f.issues.listErr = &apperrs.RetryAfterError{Wait: time.Minute, Err: errors.New("rate limited")}
	require.ErrorIs(t, f.svc.Sweep(t.Context(), true), apperrs.ErrRetryable)
	f.issues.listErr = nil
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Len(t, f.issues.queries, 1, "no listing while GitHub's wait lasts")
	f.now = f.now.Add(2 * time.Minute)
	require.NoError(t, f.svc.Sweep(t.Context(), true))
	assert.Len(t, f.issues.queries, 2)
}

func TestSweep_OneRepositoryFailing_LeavesTheOthersToRun(t *testing.T) {
	other := onSync()
	other.Repo = "zeta"
	f := newFixture(onSync(), other)
	f.issues.listErr = errors.New("boom")
	err := f.svc.Sweep(t.Context(), true)
	require.ErrorContains(t, err, "acme/app")
	assert.ErrorContains(t, err, "acme/zeta")
	assert.Len(t, f.issues.queries, 2)
}

func TestRunSweepLoop_SweepsAtStartOnCommitsAndOnTheTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFixture(onSync())
		swept(f, issueTime)
		var commits wake.Broadcast
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			f.svc.RunSweepLoop(ctx, commits.Next, slog.New(slog.DiscardHandler))
			close(done)
		}()
		synctest.Wait()
		assert.Equal(t, 1, f.issues.listed(), "every enabled sync sweeps at start")

		commits.Notify()
		synctest.Wait()
		assert.Equal(t, 1, f.issues.listed(), "a commit backfills only what is not yet backfilled")

		time.Sleep(SweepEvery)
		synctest.Wait()
		assert.Equal(t, 2, f.issues.listed(), "the timer sweeps again")

		cancel()
		<-done
	})
}
