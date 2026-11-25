package issuesync

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// stallingWriter holds one issue's update until released, as a slow GitHub response does.
type stallingWriter struct {
	IssueWriter
	number  int
	release chan struct{}
}

func (w stallingWriter) UpdateIssue(ctx context.Context, owner, repo string, number int, iw IssueWrite) (Issue, error) {
	if number == w.number {
		<-w.release
	}
	return w.IssueWriter.UpdateIssue(ctx, owner, repo, number, iw)
}

// stallingCreator holds every issue creation until released.
type stallingCreator struct {
	IssueWriter
	release chan struct{}
}

func (w stallingCreator) CreateIssue(ctx context.Context, owner, repo string, iw IssueWrite) (Issue, error) {
	<-w.release
	return w.IssueWriter.CreateIssue(ctx, owner, repo, iw)
}

func (k *keyedLocks) size() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.held)
}

func TestLocks_ASlowWriteHoldsUpOnlyItsOwnIssue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := linkedFixture(t)
		f.desk.edit("t-1", func(tk *fakeTicket) { tk.Fields.Title = "Edited in Nexul" })
		f.changed(t)
		f.now = f.now.Add(Quiet)
		release := make(chan struct{})
		f.svc.d.IssueWriter = stallingWriter{IssueWriter: f.github, number: 7, release: release}

		pushed := make(chan error, 1)
		go func() {
			_, err := f.svc.Push(t.Context())
			pushed <- err
		}()
		synctest.Wait()

		handle(t, f, "opened", newIssue(func(i *Issue) { i.Number, i.ID = 8, 9002 }))
		assert.Len(t, f.desk.tickets, 2, "#8 is filed while the write to #7 still waits on GitHub")

		same := make(chan error, 1)
		go func() {
			same <- f.svc.HandleIssue(t.Context(), "acme", "app", "edited", newIssue(withTitle("Renamed on GitHub"), later(time.Minute)), "alice-gh")
		}()
		synctest.Wait()
		select {
		case <-same:
			require.FailNow(t, "#7's webhook applied halfway through #7's write")
		default:
		}

		close(release)
		require.NoError(t, <-pushed)
		require.NoError(t, <-same)
		assert.Zero(t, f.svc.locks.size(), "no lock outlives its holders")
	})
}

func TestLocks_AnIssueBeingMadeIsAppliedOnlyOnceItsLinkIsStored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFixture(onSync())
		f.desk.tickets["t-2"] = &fakeTicket{Fields: TicketFields{Title: "Local"}}
		require.NoError(t, f.svc.TicketChanged(t.Context(), "t-2", "p-1", []string{"nexul"}))
		f.now = f.now.Add(Quiet)
		release := make(chan struct{})
		f.svc.d.IssueWriter = stallingCreator{IssueWriter: f.github, release: release}

		pushed := make(chan error, 1)
		go func() {
			_, err := f.svc.Push(t.Context())
			pushed <- err
		}()
		synctest.Wait()

		made := newIssue(func(i *Issue) { i.Number, i.ID, i.Body = 41, 9041, "Local\n\n"+ticketMarkerFor("t-2") })
		applied := make(chan error, 1)
		go func() { applied <- f.svc.HandleIssue(t.Context(), "acme", "app", "opened", made, "alice-gh") }()
		synctest.Wait()
		select {
		case <-applied:
			require.FailNow(t, "the issue was applied before the write that made it stored its link")
		default:
		}

		close(release)
		require.NoError(t, <-pushed)
		require.NoError(t, <-applied)
		assert.Len(t, f.desk.tickets, 1, "no second ticket is filed for the issue")
		assert.Equal(t, "t-2", f.repo.links[linkKey{"acme", "app", 41}].TicketID)
		assert.Zero(t, f.svc.locks.size())
	})
}

func TestKeyedLocks_DropsAKeyOnlyOnceNobodyHoldsOrWaitsOnIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var k keyedLocks
		unlock := k.lock(issueKey("Acme", "App", 7))
		waited := make(chan struct{})
		go func() {
			defer k.lock(issueKey("acme", "app", 7))()
			close(waited)
		}()
		synctest.Wait()
		select {
		case <-waited:
			require.FailNow(t, "owner and repository match in any case, so the second lock waits")
		default:
		}
		assert.Equal(t, 1, k.size())

		unlock()
		<-waited
		synctest.Wait()
		assert.Zero(t, k.size())
	})
}

// linkedLate answers the first link read as unlinked, as when the sweep links the ticket's issue just after.
type linkedLate struct {
	Repo
	reads int
}

func (r *linkedLate) GetLinkByTicket(ctx context.Context, ticketID string) (Link, error) {
	r.reads++
	if r.reads == 1 {
		return Link{}, apperrs.ErrNotFound
	}
	return r.Repo.GetLinkByTicket(ctx, ticketID)
}

func TestLockTicket_ALinkMadeMeanwhile_LocksItsIssue(t *testing.T) {
	f := linkedFixture(t)
	f.svc.d.Repo = &linkedLate{Repo: f.repo}

	unlock, link, linked, err := f.svc.lockTicket(t.Context(), "t-1")
	require.NoError(t, err)
	assert.True(t, linked)
	assert.Equal(t, 7, link.IssueNumber)
	f.svc.locks.mu.Lock()
	_, onIssue := f.svc.locks.held[issueKey("acme", "app", 7)]
	_, onTicket := f.svc.locks.held[ticketKey("t-1")]
	f.svc.locks.mu.Unlock()
	assert.True(t, onIssue, "a write to the issue holds the issue's lock")
	assert.False(t, onTicket)
	unlock()
	assert.Zero(t, f.svc.locks.size())
}
