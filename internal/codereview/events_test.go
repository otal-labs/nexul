package codereview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func openedEvent(number int) eventbus.Event {
	return eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":` + itoa(number) + `,"linked_ticket_ids":["42","43"]}}`)}
}

func TestHandlePROpened(t *testing.T) {
	t.Run("creates one pending record per pr", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))

		rs, err := s.ListByPR(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, StatusPending, rs[0].Status)
	})
	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := HandlePROpened(context.Background(), s, eventbus.Event{Payload: json.RawMessage("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("incomplete event is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := HandlePROpened(context.Background(), s, eventbus.Event{Payload: json.RawMessage(`{"pr":{"number":7}}`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("retryable on repo error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		err := HandlePROpened(context.Background(), s, openedEvent(7))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
	t.Run("duplicate event is idempotent", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		rs, err := s.ListByPR(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		assert.Len(t, rs, 1, "redelivered git.pr_opened must not duplicate the review")
	})
}

func TestHandlePRReviewSubmitted(t *testing.T) {
	t.Run("approved applies and publishes", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		ev := eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":7,"state":"approved","reviewer":"alice"}}`)}
		require.NoError(t, HandlePRReviewSubmitted(context.Background(), s, ev))

		rs, err := s.ListByPR(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, StatusApproved, rs[0].Status)
		assert.Equal(t, "alice", rs[0].Reviewer)

		b, err := json.Marshal(repo.of(TopicStatusChanged).Payload)
		require.NoError(t, err)
		var published StatusChangedEvent
		require.NoError(t, json.Unmarshal(b, &published))
		assert.Equal(t, rs[0].ID, published.ID)
		assert.Equal(t, "approved", published.Status)
	})
	t.Run("changes_requested applies", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		ev := eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":7,"state":"changes_requested","reviewer":"bob"}}`)}
		require.NoError(t, HandlePRReviewSubmitted(context.Background(), s, ev))
		rs, _ := s.ListByPR(context.Background(), "acme/app", 7)
		assert.Equal(t, StatusChangesRequested, rs[0].Status)
	})
	t.Run("commented is a no-op", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo, newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		ev := eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":7,"state":"commented","reviewer":"carol"}}`)}
		require.NoError(t, HandlePRReviewSubmitted(context.Background(), s, ev))
		rs, _ := s.ListByPR(context.Background(), "acme/app", 7)
		assert.Equal(t, StatusPending, rs[0].Status)
		assert.Empty(t, repo.of(TopicStatusChanged).Payload)
	})
	t.Run("unknown pr creates a record", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		ev := eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":99,"state":"approved","reviewer":"alice"}}`)}
		require.NoError(t, HandlePRReviewSubmitted(context.Background(), s, ev))
		rs, err := s.ListByPR(context.Background(), "acme/app", 99)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, StatusApproved, rs[0].Status)
	})
	t.Run("malformed payload is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := HandlePRReviewSubmitted(context.Background(), s, eventbus.Event{Payload: json.RawMessage("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("incomplete event is fatal", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := HandlePRReviewSubmitted(context.Background(), s, eventbus.Event{Payload: json.RawMessage(`{"pr":{"number":7,"state":"approved"}}`)})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("retryable on repo error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listPRErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		ev := eventbus.Event{Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":7,"state":"approved"}}`)}
		err := HandlePRReviewSubmitted(context.Background(), s, ev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestHandlePRMerged(t *testing.T) {
	t.Run("marks the pr review merged", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		require.NoError(t, HandlePRMerged(context.Background(), s, openedEvent(7)))

		rs, err := s.ListByPR(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, StatusMerged, rs[0].Status)
	})
	t.Run("unknown pr is a no-op", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePRMerged(context.Background(), s, openedEvent(999)))
	})
	t.Run("fatal on bad payload", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := HandlePRMerged(context.Background(), s, eventbus.Event{Payload: json.RawMessage("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("retryable on repo error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listPRErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		err := HandlePRMerged(context.Background(), s, openedEvent(7))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func TestHandlePRClosed(t *testing.T) {
	t.Run("unmerged close moves the review to closed", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePROpened(context.Background(), s, openedEvent(7)))
		require.NoError(t, HandlePRClosed(context.Background(), s, openedEvent(7)))

		rs, err := s.ListByPR(context.Background(), "acme/app", 7)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		assert.Equal(t, StatusClosed, rs[0].Status)
	})
	t.Run("unknown pr is a no-op", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		require.NoError(t, HandlePRClosed(context.Background(), s, openedEvent(999)))
	})
	t.Run("fatal on bad payload", func(t *testing.T) {
		s := newTestService(newFakeRepo(), newFakeBus())
		err := HandlePRClosed(context.Background(), s, eventbus.Event{Payload: json.RawMessage("{")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrFatal))
	})
	t.Run("retryable on repo error", func(t *testing.T) {
		repo := newFakeRepo()
		repo.listPRErr = errors.New("db down")
		s := newTestService(repo, newFakeBus())
		err := HandlePRClosed(context.Background(), s, openedEvent(7))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrRetryable))
	})
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
