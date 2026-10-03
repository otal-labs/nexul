package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/memories"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func newTestAnswer(id string, round int, question string) *memories.InterviewAnswer {
	return &memories.InterviewAnswer{
		ID: id, WorkspaceID: "workspace-default", ProjectID: "project-general", Round: round, Question: question,
		Selected: []string{"Unit"}, AnsweredBy: "user-1", AnsweredAt: time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC),
	}
}

func TestMemoriesRepo_Answers_RoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := newTestStore(t)

	last, err := s.Memories.LastRound(ctx, "project-general")
	require.NoError(t, err)
	assert.Equal(t, 0, last)

	_, err = s.Memories.UpsertAnswer(ctx, newTestAnswer("a-stack", 0, "Stack"), eventbus.OutboxEvent{ID: "evt-1", Topic: memories.TopicAnswerSaved, Payload: memories.AnswerEvent{}})
	require.NoError(t, err)
	again := newTestAnswer("a-new-id", 0, "Stack")
	again.Selected, again.Text, again.Skipped = []string{}, "", true
	saved, err := s.Memories.UpsertAnswer(ctx, again)
	require.NoError(t, err)
	assert.Equal(t, "a-stack", saved.ID, "re-answering keeps the first row")
	assert.True(t, saved.Skipped)
	assert.Empty(t, saved.Selected)

	followUp := newTestAnswer("a-cov", 1, "Coverage floor?")
	followUp.Options, followUp.MultiSelect, followUp.Why = []memories.AnswerOption{{Label: "80%", Description: "the CI gate"}}, true, "No gate in CI"
	require.NoError(t, s.Memories.InsertRound(ctx, []*memories.InterviewAnswer{followUp, newTestAnswer("a-mock", 1, "Mocks?")}))
	require.ErrorIs(t, s.Memories.InsertRound(ctx, []*memories.InterviewAnswer{newTestAnswer("a-dup", 1, "Mocks?")}), apperrs.ErrConflict)
	last, err = s.Memories.LastRound(ctx, "project-general")
	require.NoError(t, err)
	assert.Equal(t, 1, last)

	changed := newTestAnswer("ignored", 1, "Coverage floor?")
	changed.Selected, changed.Text = []string{}, "90"
	updated, err := s.Memories.UpdateAnswer(ctx, changed)
	require.NoError(t, err)
	assert.Equal(t, "90", updated.Text)
	assert.Equal(t, "No gate in CI", updated.Why, "an update keeps what was asked")
	assert.Equal(t, followUp.Options, updated.Options)
	assert.True(t, updated.MultiSelect)
	_, err = s.Memories.UpdateAnswer(ctx, newTestAnswer("x", 2, "Never asked"))
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	as, err := s.Memories.ListAnswers(ctx, "project-general")
	require.NoError(t, err)
	require.Len(t, as, 3)
	assert.Equal(t, []string{"Stack", "Coverage floor?", "Mocks?"}, []string{as[0].Question, as[1].Question, as[2].Question})

	require.NoError(t, s.Memories.DeleteAnswer(ctx, "project-general", 0, "Stack"))
	require.ErrorIs(t, s.Memories.DeleteAnswer(ctx, "project-general", 0, "Stack"), apperrs.ErrNotFound)
}

func TestMemoriesRepo_Answers_UnknownProjectIsAConflict(t *testing.T) {
	t.Parallel()
	a := newTestAnswer("a-1", 0, "Stack")
	a.ProjectID = "project-missing"
	_, err := newTestStore(t).Memories.UpsertAnswer(context.Background(), a)
	require.ErrorIs(t, err, apperrs.ErrConflict)
}
