package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/plays"
)

func newTestTrail(id string, startedAt time.Time) *plays.Trail {
	return &plays.Trail{
		ID: id, WorkspaceID: "ws-1", PlayID: "play-1", PlayLabel: "Fix with AI", TargetType: plays.TargetTicket,
		TargetID: "t-1", ProjectID: "proj-1", ConversationID: "conv-1", StarterID: "u-1", Via: plays.ViaWeb,
		SelectedMemoryIDs: []string{"m-2", "m-1"}, CustomInstructions: "careful", MoveToStatusID: "st-review",
		State: plays.TrailStarting, StartedAt: startedAt, Activity: []plays.ActivityEntry{},
	}
}

func newTrailStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	require.NoError(t, s.Workspaces.Create(context.Background(), newTestWorkspace("ws-1", "Acme")))
	return s
}

func TestPlayTrailsRepo_Create_Get_RoundTrip(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	startedAt := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), newTestTrail("tr-1", startedAt)))

	got, err := s.PlayTrails.GetTrail(context.Background(), "tr-1")
	require.NoError(t, err)
	assert.Equal(t, "Fix with AI", got.PlayLabel)
	assert.Equal(t, plays.TargetTicket, got.TargetType)
	assert.Equal(t, plays.ViaWeb, got.Via)
	assert.Equal(t, []string{"m-2", "m-1"}, got.SelectedMemoryIDs, "the repo keeps the order it was handed")
	assert.Equal(t, "careful", got.CustomInstructions)
	assert.Equal(t, "st-review", got.MoveToStatusID)
	assert.Equal(t, plays.TrailStarting, got.State)
	assert.Equal(t, startedAt, got.StartedAt)
	assert.Nil(t, got.EndedAt)
	assert.Equal(t, []plays.ActivityEntry{}, got.Activity)
	assert.Empty(t, got.HarnessSessionID)
}

func TestPlayTrailsRepo_Create_EmptyMoveTo_IsNull(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	tr := newTestTrail("tr-1", time.Now())
	tr.MoveToStatusID = ""
	tr.SelectedMemoryIDs = nil
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), tr))

	got, err := s.PlayTrails.GetTrail(context.Background(), "tr-1")
	require.NoError(t, err)
	assert.Empty(t, got.MoveToStatusID)
	assert.Equal(t, []string{}, got.SelectedMemoryIDs)
}

func TestPlayTrailsRepo_Create_Duplicate_Conflicts(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), newTestTrail("tr-1", time.Now())))
	err := s.PlayTrails.CreateTrail(context.Background(), newTestTrail("tr-1", time.Now()))
	require.ErrorIs(t, err, apperrs.ErrConflict)
}

func TestPlayTrailsRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	_, err := s.PlayTrails.GetTrail(context.Background(), "missing")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlayTrailsRepo_Update_PersistsTheRunsMovingParts(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	tr := newTestTrail("tr-1", time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC))
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), tr))

	ended := time.Date(2026, 9, 17, 9, 5, 0, 0, time.UTC)
	tr.State = plays.TrailDone
	tr.HarnessSessionID = "sess-1"
	tr.EndedAt = &ended
	tr.LastError = ""
	tr.ReplyMessageID = "msg-9"
	tr.Activity = []plays.ActivityEntry{{Kind: harness.ActivityToolResult, CallID: "c-1", Tool: "Read", Summary: "Read main.go", Detail: `{"input":{"file_path":"main.go"}}`, At: ended}, {Kind: harness.ActivityText, Summary: "Bash go test", At: ended}}
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: plays.TopicRunFinished, Payload: map[string]any{"outcome": "done"}}
	require.NoError(t, s.PlayTrails.UpdateTrail(context.Background(), tr, evt))

	pending, err := s.Outbox.Unpublished(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, pending, 1, "the run event rides in the trail update's transaction")
	assert.Equal(t, plays.TopicRunFinished, pending[0].Topic)

	got, err := s.PlayTrails.GetTrail(context.Background(), "tr-1")
	require.NoError(t, err)
	assert.Equal(t, plays.TrailDone, got.State)
	assert.Equal(t, "sess-1", got.HarnessSessionID)
	require.NotNil(t, got.EndedAt)
	assert.Equal(t, ended, *got.EndedAt)
	assert.Equal(t, "msg-9", got.ReplyMessageID)
	assert.Equal(t, tr.Activity, got.Activity, "steps round-trip whole, times in UTC")
	assert.Equal(t, "careful", got.CustomInstructions, "the press-time fields are untouched")
}

func TestPlayTrailsRepo_Update_QuestionRoundTripsAndClears(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	tr := newTestTrail("tr-1", time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC))
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), tr))
	got, err := s.PlayTrails.GetTrail(context.Background(), "tr-1")
	require.NoError(t, err)
	assert.Nil(t, got.Question, "a run that never asked stores NULL")

	asked := time.Date(2026, 9, 17, 9, 3, 0, 0, time.UTC)
	tr.State = plays.TrailWaiting
	tr.Question = &plays.TrailQuestion{
		Question: harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{ID: "q1", Text: "Proceed?", Header: "Nexul", Options: []harness.QuestionOption{{Label: "Yes", Description: "go on"}, {Label: "No"}}}}},
		AskedAt:  asked,
	}
	require.NoError(t, s.PlayTrails.UpdateTrail(context.Background(), tr))
	got, err = s.PlayTrails.GetTrail(context.Background(), "tr-1")
	require.NoError(t, err)
	assert.Equal(t, plays.TrailWaiting, got.State)
	assert.Equal(t, tr.Question, got.Question)

	active, err := s.PlayTrails.ListActiveTrailsByTargets(context.Background(), plays.TargetTicket, []string{"t-1"})
	require.NoError(t, err)
	assert.Len(t, active, 1, "a waiting run still occupies its target")

	tr.Question.Answer = &harness.QuestionAnswer{Answers: map[string]harness.AnswerValue{"q1": {Selected: []string{"Yes"}}}}
	tr.State = plays.TrailRunning
	require.NoError(t, s.PlayTrails.UpdateTrail(context.Background(), tr))
	got, err = s.PlayTrails.GetTrail(context.Background(), "tr-1")
	require.NoError(t, err)
	require.NotNil(t, got.Question.Answer)
	assert.Equal(t, []string{"Yes"}, got.Question.Answer.Answers["q1"].Selected)
}

func TestPlayTrailsRepo_Get_LegacyLineActivity_ReadsAsOtherSteps(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), newTestTrail("tr-old", time.Now())))
	_, err := s.db.ExecContext(context.Background(), `UPDATE play_trails SET activity = ? WHERE id = ?`, `["Read main.go","Bash go test"]`, "tr-old")
	require.NoError(t, err)

	got, err := s.PlayTrails.GetTrail(context.Background(), "tr-old")
	require.NoError(t, err)
	assert.Equal(t, []plays.ActivityEntry{
		{Kind: harness.ActivityOther, Summary: "Read main.go"},
		{Kind: harness.ActivityOther, Summary: "Bash go test"},
	}, got.Activity)
}

func TestPlayTrailsRepo_Update_Missing_NotFound(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	err := s.PlayTrails.UpdateTrail(context.Background(), newTestTrail("ghost", time.Now()))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlayTrailsRepo_ListByTarget_NewestFirst(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	base := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), newTestTrail("older", base)))
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), newTestTrail("newer", base.Add(time.Minute))))
	other := newTestTrail("other-target", base.Add(2*time.Minute))
	other.TargetID = "t-2"
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), other))

	list, err := s.PlayTrails.ListTrailsByTarget(context.Background(), plays.TargetTicket, "t-1")
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "newer", list[0].ID)
	assert.Equal(t, "older", list[1].ID)

	none, err := s.PlayTrails.ListTrailsByTarget(context.Background(), plays.TargetDoc, "t-1")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestPlayTrailsRepo_LatestForChoices(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	_, err := s.PlayTrails.LatestTrailForChoices(context.Background(), "u-1", "play-1", "proj-1")
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	base := time.Date(2026, 9, 17, 9, 0, 0, 0, time.UTC)
	older := newTestTrail("older", base)
	older.MoveToStatusID = "st-old"
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), older))
	newer := newTestTrail("newer", base.Add(time.Minute))
	newer.MoveToStatusID = "st-new"
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), newer))
	someoneElse := newTestTrail("theirs", base.Add(time.Hour))
	someoneElse.StarterID = "u-2"
	require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), someoneElse))

	got, err := s.PlayTrails.LatestTrailForChoices(context.Background(), "u-1", "play-1", "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "newer", got.ID)
	assert.Equal(t, "st-new", got.MoveToStatusID)
}

func TestPlayTrailsRepo_ListActiveByTargets(t *testing.T) {
	t.Parallel()
	s := newTrailStore(t)
	now := time.Now()
	running := newTestTrail("running", now)
	running.State = plays.TrailRunning
	done := newTestTrail("done", now)
	done.TargetID = "t-2"
	done.State = plays.TrailDone
	starting := newTestTrail("starting", now)
	starting.TargetID = "t-3"
	for _, tr := range []*plays.Trail{running, done, starting} {
		require.NoError(t, s.PlayTrails.CreateTrail(context.Background(), tr))
	}

	got, err := s.PlayTrails.ListActiveTrailsByTargets(context.Background(), plays.TargetTicket, []string{"t-1", "t-2", "t-3", "t-4"})
	require.NoError(t, err)
	ids := make([]string, 0, len(got))
	for _, tr := range got {
		ids = append(ids, tr.ID)
	}
	assert.ElementsMatch(t, []string{"running", "starting"}, ids)

	none, err := s.PlayTrails.ListActiveTrailsByTargets(context.Background(), plays.TargetTicket, nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}
