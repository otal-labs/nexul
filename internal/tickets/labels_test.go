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

func TestAddLabel(t *testing.T) {
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
	s := newTestService(newFakeRepo())
	for project, label := range map[string]string{"p-1": "bug", "p-2": "chore"} {
		created, err := s.Create(context.Background(), project, "a", "", "", "")
		require.NoError(t, err)
		_, err = s.AddLabel(context.Background(), created.ID, label)
		require.NoError(t, err)
	}
	t.Run("lists one project's labels", func(t *testing.T) {
		labels, err := s.ListAllLabels(context.Background(), "p-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"bug"}, labels)
	})
	t.Run("lists every readable label without a project", func(t *testing.T) {
		labels, err := s.ListAllLabels(context.Background(), "")
		require.NoError(t, err)
		assert.Equal(t, []string{"bug", "chore"}, labels)
	})
}

func TestSetLabelColor(t *testing.T) {
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
}

func TestLabelColors(t *testing.T) {
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
}

func TestUpdateStatus_ConfiguredOnly(t *testing.T) {
	t.Run("status store error is retried", func(t *testing.T) {
		repo := newFakeRepo()
		s := NewService(repo, fakeStatusStore{err: errors.New("db down")}, nil)
		s.now = func() time.Time { return fixedNow }
		created, err := s.Create(context.Background(), "p-1", "ticket", "", "", "")
		require.NoError(t, err)
		_, err = s.UpdateStatus(context.Background(), created.ID, StatusDone)
		require.Error(t, err)
		assert.ErrorContains(t, err, "db down")
	})
}

func TestLabelRefusals(t *testing.T) {
	ctx := context.Background()
	errDown := errors.New("db down")
	tests := []struct {
		name string
		seed func(repo *fakeRepo)
		call func(s *Service) error
		want error
	}{
		{"set a type without a ticket", nil, func(s *Service) error { _, err := s.SetType(ctx, "", "t-1"); return err }, apperrs.ErrInvalid},
		{"set a blank type", nil, func(s *Service) error { _, err := s.SetType(ctx, "t-1", "  "); return err }, apperrs.ErrInvalid},
		{"set a type on a missing ticket", nil, func(s *Service) error { _, err := s.SetType(ctx, "nope", "t-1"); return err }, apperrs.ErrNotFound},
		{"add a label without a ticket", nil, func(s *Service) error { _, err := s.AddLabel(ctx, "", "bug"); return err }, apperrs.ErrInvalid},
		{"add a blank label", nil, func(s *Service) error { _, err := s.AddLabel(ctx, "t-1", "  "); return err }, apperrs.ErrInvalid},
		{"add a label to a missing ticket", nil, func(s *Service) error { _, err := s.AddLabel(ctx, "nope", "bug"); return err }, apperrs.ErrNotFound},
		{"remove a blank label", nil, func(s *Service) error { _, err := s.RemoveLabel(ctx, "t-1", "  "); return err }, apperrs.ErrInvalid},
		{"remove a label from a missing ticket", nil, func(s *Service) error { _, err := s.RemoveLabel(ctx, "nope", "bug"); return err }, apperrs.ErrNotFound},
		{"list the labels of a blank ticket", nil, func(s *Service) error { _, err := s.ListLabels(ctx, " "); return err }, apperrs.ErrInvalid},
		{"list the labels of a missing ticket", nil, func(s *Service) error { _, err := s.ListLabels(ctx, "nope"); return err }, apperrs.ErrNotFound},
		{"color a label without a project", nil, func(s *Service) error { _, err := s.SetLabelColor(ctx, "  ", "bug", colors.Cyan); return err }, apperrs.ErrInvalid},
		{"color a blank label", nil, func(s *Service) error { _, err := s.SetLabelColor(ctx, "p-1", "  ", colors.Cyan); return err }, apperrs.ErrInvalid},
		{"color a label with a color that is not one", nil, func(s *Service) error {
			_, err := s.SetLabelColor(ctx, "p-1", "bug", colors.Color("magenta"))
			return err
		}, apperrs.ErrInvalid},
		{"color a label with no color", nil, func(s *Service) error { _, err := s.SetLabelColor(ctx, "p-1", "bug", colors.Color("")); return err }, apperrs.ErrInvalid},
		{"color a label when the repo fails", func(r *fakeRepo) { r.labelColorErr = errDown }, func(s *Service) error {
			_, err := s.SetLabelColor(ctx, "p-1", "bug", colors.Cyan)
			return err
		}, errDown},
		{"read label colors without a project", nil, func(s *Service) error { _, err := s.LabelColors(ctx, " ", []string{"bug"}); return err }, apperrs.ErrInvalid},
		{"read label colors when the repo fails", func(r *fakeRepo) { r.labelColorErr = errDown }, func(s *Service) error {
			_, err := s.LabelColors(ctx, "p-1", []string{"bug"})
			return err
		}, errDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tt.seed != nil {
				tt.seed(repo)
			}
			require.ErrorIs(t, tt.call(newTestService(repo)), tt.want)
		})
	}
}
