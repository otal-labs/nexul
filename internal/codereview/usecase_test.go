package codereview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

var fixedNow = time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

// fakeRepo is an in-memory codereview.Repo for use-case tests.
type fakeRepo struct {
	mu          sync.Mutex
	reviews     map[string]*CodeReview
	ticketLinks map[string][]string // ticket id -> "repo#number" PR keys
	outbox      []eventbus.OutboxEvent
	createErr   error
	getErr      error
	statusErr   error
	listErr     error
	listPRErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{reviews: map[string]*CodeReview{}, ticketLinks: map[string][]string{}}
}

// link models the tickets-domain ticket_pr_links join for ListByTicket.
func (f *fakeRepo) link(ticketID, repo string, number int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ticketLinks[ticketID] = append(f.ticketLinks[ticketID], fmt.Sprintf("%s#%d", repo, number))
}

func (f *fakeRepo) Create(_ context.Context, r *CodeReview) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	for _, existing := range f.reviews {
		if existing.Repo == r.Repo && existing.PRNumber == r.PRNumber {
			return nil
		}
	}
	f.reviews[r.ID] = r
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (*CodeReview, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	r, ok := f.reviews[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return r, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id string, status Status, reviewer string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.statusErr != nil {
		return f.statusErr
	}
	r, ok := f.reviews[id]
	if !ok {
		return apperrs.ErrNotFound
	}
	r.Status = status
	r.Reviewer = reviewer
	f.outbox = append(f.outbox, evts...)
	return nil
}

// of returns the last outbox event enqueued for a topic.
func (f *fakeRepo) of(topic string) eventbus.OutboxEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	var evt eventbus.OutboxEvent
	for _, e := range f.outbox {
		if e.Topic == topic {
			evt = e
		}
	}
	return evt
}

// count returns how many outbox events were enqueued for a topic.
func (f *fakeRepo) count(topic string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, e := range f.outbox {
		if e.Topic == topic {
			n++
		}
	}
	return n
}

func (f *fakeRepo) ListByTicket(_ context.Context, ticketID string) ([]*CodeReview, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*CodeReview
	for _, key := range f.ticketLinks[ticketID] {
		for _, r := range f.reviews {
			if key == fmt.Sprintf("%s#%d", r.Repo, r.PRNumber) {
				out = append(out, r)
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) ListByPR(_ context.Context, repo string, number int) ([]*CodeReview, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listPRErr != nil {
		return nil, f.listPRErr
	}
	var out []*CodeReview
	for _, r := range f.reviews {
		if r.Repo == repo && r.PRNumber == number {
			out = append(out, r)
		}
	}
	return out, nil
}

// fakeBus records published events, mirroring the deploy/topology fakes.
type fakeBus struct {
	mu         sync.Mutex
	published  []eventbus.Event
	publishErr error
}

func newFakeBus() *fakeBus {
	return &fakeBus{}
}

func (f *fakeBus) Publish(_ context.Context, topic string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.publishErr != nil {
		return f.publishErr
	}
	b, _ := json.Marshal(payload)
	f.published = append(f.published, eventbus.Event{Topic: topic, Payload: b})
	return nil
}

// bus is accepted for call-site compatibility but unused: review.status_changed reaches the bus via the transactional outbox on repo, not direct Publish; tests assert on repo.of(...)/repo.count(...) instead.
func newTestService(repo *fakeRepo, bus *fakeBus) *Service {
	s := NewService(repo)
	s.now = func() time.Time { return fixedNow }
	return s
}

func mustCreate(t *testing.T, s *Service, repo string, number int) *CodeReview {
	t.Helper()
	r, err := s.Create(context.Background(), repo, number)
	require.NoError(t, err)
	return r
}

func TestCreate(t *testing.T) {
	t.Run("blank repo is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Create(context.Background(), "  ", 1)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-positive number is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Create(context.Background(), "acme/app", 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		_, err := s.Create(context.Background(), "acme/app", 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.createErr)
	})
	t.Run("creates a pending review", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		r, err := s.Create(context.Background(), "acme/app", 42)
		require.NoError(t, err)
		assert.Equal(t, StatusPending, r.Status)
		assert.Equal(t, "acme/app", r.Repo)
		assert.Equal(t, 42, r.PRNumber)
		assert.Equal(t, fixedNow, r.CreatedAt)
	})
}

func TestApplyReviewSubmitted(t *testing.T) {
	t.Run("blank repo is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.ApplyReviewSubmitted(context.Background(), " ", 1, "approved", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-positive number is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 0, "approved", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("approved flips the status and records the reviewer", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		mustCreate(t, s, "acme/app", 7)
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "approved", "alice")
		require.NoError(t, err)
		assert.Equal(t, StatusApproved, got.Status)
		assert.Equal(t, "alice", got.Reviewer)

		b, err := json.Marshal(repo.of(TopicStatusChanged).Payload)
		require.NoError(t, err)
		var ev StatusChangedEvent
		require.NoError(t, json.Unmarshal(b, &ev))
		assert.Equal(t, "acme/app", ev.Repo)
		assert.Equal(t, 7, ev.PRNumber)
		assert.Equal(t, string(StatusApproved), ev.Status)
		assert.Equal(t, "alice", ev.Reviewer)
	})
	t.Run("changes_requested flips the status", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		created := mustCreate(t, s, "acme/app", 7)
		require.NoError(t, s.repo.(*fakeRepo).UpdateStatus(context.Background(), created.ID, StatusApproved, ""))
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "changes_requested", "bob")
		require.NoError(t, err)
		assert.Equal(t, StatusChangesRequested, got.Status)
	})
	t.Run("latest review wins", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		created := mustCreate(t, s, "acme/app", 7)
		require.NoError(t, s.repo.(*fakeRepo).UpdateStatus(context.Background(), created.ID, StatusChangesRequested, ""))
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "approved", "alice")
		require.NoError(t, err)
		assert.Equal(t, StatusApproved, got.Status)
	})
	t.Run("commented review does not override an approval", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		created := mustCreate(t, s, "acme/app", 7)
		require.NoError(t, repo.UpdateStatus(context.Background(), created.ID, StatusApproved, "alice"))
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "commented", "carol")
		require.NoError(t, err)
		assert.Equal(t, StatusApproved, got.Status)
		assert.Equal(t, "alice", got.Reviewer)
		assert.Empty(t, repo.of(TopicStatusChanged).Payload, "no event for a no-op review")
	})
	t.Run("dismissed review does not flip the status", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		mustCreate(t, s, "acme/app", 7)
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "dismissed", "dan")
		require.NoError(t, err)
		assert.Equal(t, StatusPending, got.Status)
	})
	t.Run("unknown pr creates a record from the event", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 99, "approved", "alice")
		require.NoError(t, err)
		assert.Equal(t, StatusApproved, got.Status)
		assert.Equal(t, "alice", got.Reviewer)
	})
	t.Run("same status is a no-op", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		created := mustCreate(t, s, "acme/app", 7)
		require.NoError(t, repo.UpdateStatus(context.Background(), created.ID, StatusApproved, "alice"))
		got, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "approved", "alice")
		require.NoError(t, err)
		assert.Equal(t, StatusApproved, got.Status)
		assert.Empty(t, repo.of(TopicStatusChanged).Payload, "no event for a same-status review")
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listPRErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		_, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "approved", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listPRErr)
	})
	t.Run("outbox enqueue error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		mustCreate(t, newTestService(repo, newFakeBus()), "acme/app", 7)
		repo.statusErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		_, err := s.ApplyReviewSubmitted(context.Background(), "acme/app", 7, "approved", "alice")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.statusErr)
	})
}

func TestApplyPRMerged(t *testing.T) {
	t.Run("marks the pr review merged", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		mustCreate(t, s, "acme/app", 7)
		got, err := s.ApplyPRMerged(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, StatusMerged, got.Status)
	})
	t.Run("unknown pr is a no-op", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		got, err := s.ApplyPRMerged(context.Background(), "acme/app", 999)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("merged stays merged", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		mustCreate(t, s, "acme/app", 7)
		_, err := s.ApplyPRMerged(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		_, err = s.ApplyPRMerged(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.count(TopicStatusChanged), "redelivered merged event enqueues nothing new")
	})
}

func TestApplyPRClosed(t *testing.T) {
	t.Run("unmerged close moves the review to closed", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		mustCreate(t, s, "acme/app", 7)
		got, err := s.ApplyPRClosed(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, StatusClosed, got.Status)
	})
	t.Run("closed review stays closed", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		mustCreate(t, s, "acme/app", 7)
		_, err := s.ApplyPRClosed(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		_, err = s.ApplyPRClosed(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		assert.Equal(t, 1, repo.count(TopicStatusChanged), "redelivered closed event enqueues nothing new")
	})
	t.Run("unknown pr is a no-op", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		got, err := s.ApplyPRClosed(context.Background(), "acme/app", 999)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestGet(t *testing.T) {
	t.Run("blank id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Get(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing review is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.Get(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.getErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		_, err := s.Get(context.Background(), "r-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.getErr)
	})
	t.Run("returns a stored review", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		created := mustCreate(t, s, "acme/app", 42)
		got, err := s.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
}

func TestListByPR(t *testing.T) {
	t.Run("blank repo is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.ListByPR(context.Background(), " ", 42)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("non-positive number is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		_, err := s.ListByPR(context.Background(), "acme/app", 0)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listPRErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		_, err := s.ListByPR(context.Background(), "acme/app", 42)
		require.Error(t, err)
		assert.ErrorIs(t, err, repo.listPRErr)
	})
	t.Run("one record per pr", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		mustCreate(t, s, "acme/app", 42)
		mustCreate(t, s, "acme/app", 43)
		mustCreate(t, s, "other/app", 42)
		rs, err := s.ListByPR(context.Background(), "acme/app", 42)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, 42, rs[0].PRNumber)
	})
}
