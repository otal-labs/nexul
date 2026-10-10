package docs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/wake"
)

// fakeSettles is a SettleRepo over open windows a test opens; settling a window commits, as the real one does.
type fakeSettles struct {
	mu      sync.Mutex
	open    map[string]time.Time
	failing int
	reads   int
	events  []eventbus.OutboxEvent
	commits *wake.Broadcast
}

func newFakeSettles() *fakeSettles {
	return &fakeSettles{open: map[string]time.Time{}, commits: &wake.Broadcast{}}
}

func (f *fakeSettles) NextSettleDue(context.Context) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	var next time.Time
	for _, due := range f.open {
		if next.IsZero() || due.Before(next) {
			next = due
		}
	}
	return next, !next.IsZero(), nil
}

func (f *fakeSettles) SettleDue(_ context.Context, now time.Time, build func([]SettledEvent) []eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failing > 0 {
		f.failing--
		return errors.New("database is locked")
	}
	var settled []SettledEvent
	for id, due := range f.open {
		if !due.After(now) {
			settled = append(settled, SettledEvent{Doc: WatchedDoc{ID: id}, ActorID: "u-1"})
			delete(f.open, id)
		}
	}
	f.events = append(f.events, build(settled)...)
	f.commits.Notify()
	return nil
}

// save opens or moves a doc's window the way a person's save commits it.
func (f *fakeSettles) save(id string) {
	f.mu.Lock()
	f.open[id] = time.Now().Add(SettleWindow)
	f.mu.Unlock()
	f.commits.Notify()
}

func (f *fakeSettles) settledIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.events))
	for _, e := range f.events {
		out = append(out, e.Payload.(SettledEvent).Doc.ID)
	}
	return out
}

func (f *fakeSettles) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

// startSettleLoop runs the loop until the test ends and checks it returns once its context does.
func startSettleLoop(t *testing.T, f *fakeSettles) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		RunSettleLoop(ctx, f, f.commits.Next, slog.New(slog.DiscardHandler))
		close(stopped)
	}()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	synctest.Wait()
}

func TestRunSettleLoop_AFailedSettleIsRetried(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFakeSettles()
		f.failing = 1
		f.open["d-1"] = time.Now()
		startSettleLoop(t, f)
		assert.Empty(t, f.settledIDs())

		time.Sleep(settleRetry)
		synctest.Wait()
		assert.Equal(t, []string{"d-1"}, f.settledIDs())
	})
}

func TestRunSettleLoop_AWindowDueWhileDownSettlesOnStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFakeSettles()
		f.open["d-1"] = time.Now().Add(-time.Hour)
		startSettleLoop(t, f)
		require.Len(t, f.events, 1)
		assert.Equal(t, TopicSettled, f.events[0].Topic)
		assert.Equal(t, []string{"d-1"}, f.settledIDs())
	})
}

func TestRunSettleLoop_SettlesOnlyOnceEditsStopForTheWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFakeSettles()
		startSettleLoop(t, f)
		f.save("d-1")
		synctest.Wait()

		time.Sleep(SettleWindow - time.Minute)
		f.save("d-1")
		synctest.Wait()
		time.Sleep(SettleWindow - time.Second)
		synctest.Wait()
		assert.Empty(t, f.settledIDs(), "an edit inside the window moves it")

		time.Sleep(time.Second)
		synctest.Wait()
		assert.Equal(t, []string{"d-1"}, f.settledIDs())
	})
}

func TestRunSettleLoop_NothingOpenReadsOnlyOnCommits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFakeSettles()
		startSettleLoop(t, f)
		before := f.readCount()

		time.Sleep(24 * time.Hour)
		synctest.Wait()
		assert.Equal(t, before, f.readCount(), "an idle loop polls nothing")
	})
}

func TestSettler_OnlyAPersonsOwnEditOpensTheWindow(t *testing.T) {
	automation := identity.WithActor(context.Background(), identity.Actor{ID: "user-1", Automation: &identity.AutomationRef{ID: "auto-1"}})
	tests := []struct {
		name string
		save func(t *testing.T, s *Service, d *Doc) error
		want string
	}{
		{"a person's edit", func(_ *testing.T, s *Service, d *Doc) error {
			_, err := s.Update(testCtx(), d.ID, "Renamed", d.Body)
			return err
		}, "user-1"},
		{"a person's live editor commit", func(_ *testing.T, s *Service, d *Doc) error {
			return s.CommitCollab(testCtx(), d.ID, "Renamed", d.Body)
		}, "user-1"},
		{"a person's save that changes nothing", func(_ *testing.T, s *Service, d *Doc) error {
			_, err := s.Update(testCtx(), d.ID, d.Title, d.Body)
			return err
		}, ""},
		{"an agent's edit over MCP", func(t *testing.T, s *Service, d *Doc) error {
			_, err := callTool(testCtx(), t, s, "doc_update", `{"id":"`+d.ID+`","title":"Renamed"}`)
			return err
		}, ""},
		{"an automation's edit", func(_ *testing.T, s *Service, d *Doc) error {
			_, err := s.Update(automation, d.ID, "Renamed", d.Body)
			return err
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			s := newTestService(repo)
			d, err := s.Create(testCtx(), "project-1", "Spec", "**bold**")
			require.NoError(t, err)
			require.Equal(t, "user-1", repo.docs[d.ID].Settler, "a person's create opens the window")

			require.NoError(t, tt.save(t, s, d))
			assert.Equal(t, tt.want, repo.docs[d.ID].Settler)
		})
	}
}

func TestSettler_AnAgentsCreateOverMCPOpensNoWindow(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	_, err := callTool(testCtx(), t, s, "doc_create", `{"project_id":"project-1","title":"Spec","body":"draft"}`)
	require.NoError(t, err)
	require.Len(t, repo.docs, 1)
	for _, d := range repo.docs {
		assert.Empty(t, d.Settler)
	}
}
