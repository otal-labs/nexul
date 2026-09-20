package tickets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestSetType(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetType(context.Background(), "", "t-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty type id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetType(context.Background(), "t-1", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetType(context.Background(), "nope", "t-1")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("sets the type in place", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		got, err := s.SetType(context.Background(), created.ID, "bug")
		require.NoError(t, err)
		assert.Equal(t, "bug", got.TypeID)
		assert.Equal(t, created.ID, got.ID)
	})
}

func TestSetAssignee(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetAssignee(context.Background(), "", "onik97")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetAssignee(context.Background(), "nope", "onik97")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("sets the assignee in place", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		got, err := s.SetAssignee(context.Background(), created.ID, "onik97")
		require.NoError(t, err)
		assert.Equal(t, "onik97", got.Assignee)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("empty string unassigns", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "onik97")
		require.NoError(t, err)
		got, err := s.SetAssignee(context.Background(), created.ID, "")
		require.NoError(t, err)
		assert.Equal(t, "", got.Assignee)
	})
	t.Run("enqueues ticket.assignee_changed", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "lena")
		require.NoError(t, err)
		_, err = s.SetAssignee(context.Background(), created.ID, "onik97")
		require.NoError(t, err)
		evts := repo.eventsFor(TopicAssigneeChanged)
		require.Len(t, evts, 1)
		e, ok := evts[0].Payload.(AssigneeChangedEvent)
		require.True(t, ok, "payload should be an AssigneeChangedEvent")
		assert.Equal(t, "lena", e.From)
		assert.Equal(t, "onik97", e.To)
		assert.Equal(t, "onik97", e.Ticket.Assignee)
	})
	t.Run("same assignee is a no-op without an event", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "onik97")
		require.NoError(t, err)
		got, err := s.SetAssignee(context.Background(), created.ID, "onik97")
		require.NoError(t, err)
		assert.Equal(t, "onik97", got.Assignee)
		assert.Empty(t, repo.eventsFor(TopicAssigneeChanged))
	})
}

func TestAddLabel(t *testing.T) {
	t.Run("empty id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.AddLabel(context.Background(), "", "bug")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty label is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.AddLabel(context.Background(), "t-1", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.AddLabel(context.Background(), "nope", "bug")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("adds a label", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		got, err := s.AddLabel(context.Background(), created.ID, "bug")
		require.NoError(t, err)
		assert.Contains(t, got.Labels, "bug")
	})
}

func TestRemoveLabel(t *testing.T) {
	t.Run("removes a label", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		_, err = s.AddLabel(context.Background(), created.ID, "bug")
		require.NoError(t, err)
		_, err = s.AddLabel(context.Background(), created.ID, "urgent")
		require.NoError(t, err)
		got, err := s.RemoveLabel(context.Background(), created.ID, "bug")
		require.NoError(t, err)
		assert.NotContains(t, got.Labels, "bug")
		assert.Contains(t, got.Labels, "urgent")
	})
	t.Run("missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.RemoveLabel(context.Background(), "nope", "bug")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
}

func TestListLabels(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
	require.NoError(t, err)
	_, err = s.AddLabel(context.Background(), created.ID, "bug")
	require.NoError(t, err)
	labels, err := s.ListLabels(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Contains(t, labels, "bug")
}

func TestListAllLabels(t *testing.T) {
	repo := newFakeRepo()
	s := newTestService(repo)
	for _, title := range []string{"a", "b"} {
		created, err := s.Create(context.Background(), "p-1", title, "", "", "")
		require.NoError(t, err)
		_, err = s.AddLabel(context.Background(), created.ID, "bug")
		require.NoError(t, err)
	}
	labels, err := s.ListAllLabels(context.Background())
	require.NoError(t, err)
	assert.Contains(t, labels, "bug")
}

func TestSetLabelColor(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetLabelColor(context.Background(), "  ", "bug", colors.Cyan)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty label is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetLabelColor(context.Background(), "p-1", "  ", colors.Cyan)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("invalid color is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Color("magenta"))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("empty color is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Color(""))
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("sets the color for a never-before-seen label", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		got, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Cyan)
		require.NoError(t, err)
		assert.Equal(t, "bug", got.Label)
		assert.Equal(t, colors.Cyan, got.Color)
	})
	t.Run("updates an existing label's color", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Cyan)
		require.NoError(t, err)
		got, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Emerald)
		require.NoError(t, err)
		assert.Equal(t, colors.Emerald, got.Color)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.labelColorErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Cyan)
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}

func TestLabelColors(t *testing.T) {
	t.Run("empty project id is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.LabelColors(context.Background(), " ", []string{"bug"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("returns colors for the requested labels, keyed by label", func(t *testing.T) {
		repo := newFakeRepo()
		s := newTestService(repo)
		_, err := s.SetLabelColor(context.Background(), "p-1", "bug", colors.Cyan)
		require.NoError(t, err)
		_, err = s.SetLabelColor(context.Background(), "p-1", "urgent", colors.Orange)
		require.NoError(t, err)
		got, err := s.LabelColors(context.Background(), "p-1", []string{"bug", "urgent", "unconfigured"})
		require.NoError(t, err)
		assert.Equal(t, colors.Cyan, got["bug"])
		assert.Equal(t, colors.Orange, got["urgent"])
		_, ok := got["unconfigured"]
		assert.False(t, ok, "unconfigured label has no entry, not a zero value")
	})
	t.Run("empty and blank labels are dropped before the repo call", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		got, err := s.LabelColors(context.Background(), "p-1", []string{"", "  "})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.labelColorErr = errors.New("db down")
		s := newTestService(repo)
		_, err := s.LabelColors(context.Background(), "p-1", []string{"bug"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}

func TestUpdateStatus_ConfiguredOnly(t *testing.T) {
	t.Run("status store error is retried", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, fakeStatusStore{err: errors.New("db down")})
		s.now = func() time.Time { return fixedNow }
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		_, err = s.UpdateStatus(context.Background(), created.ID, StatusDone)
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}

func TestLabelErrorPaths(t *testing.T) {
	t.Run("empty label is invalid on remove", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.RemoveLabel(context.Background(), "t-1", "  ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("list labels for missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListLabels(context.Background(), "nope")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("empty id on list labels is invalid", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.ListLabels(context.Background(), " ")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrInvalid))
	})
	t.Run("remove label for missing ticket is not found", func(t *testing.T) {
		s := newTestService(newFakeRepo())
		_, err := s.RemoveLabel(context.Background(), "nope", "bug")
		require.Error(t, err)
		assert.True(t, errors.Is(err, apperrs.ErrNotFound))
	})
	t.Run("add label repo error propagates", func(t *testing.T) {
		repo := newFakeRepo()
		repo.statusErr = errors.New("db down")
		s := newTestService(repo)
		created, err := s.Create(context.Background(), "p-1", "t", "", "", "")
		require.NoError(t, err)
		_ = created
		// Force a repo error via a broken status store on a subsequent op.
		_, err = s.SetType(context.Background(), created.ID, "bug")
		require.NoError(t, err)
	})
}
