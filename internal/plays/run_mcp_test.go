package plays

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestRunMCPTools_Surface(t *testing.T) {
	tools := RunMCPTools(newRunnerFixture().runner)
	assert.ElementsMatch(t, []string{"play_run", "trail_list", "trail_update", "decisions_check_run"}, toolNames(tools))
	for _, tool := range tools {
		assert.NotEmpty(t, tool.Title, tool.Name)
	}
}

func TestRunMCPTools_ErrorPaths(t *testing.T) {
	tests := []struct {
		name  string
		actor string
		tool  string
		args  string
		want  error
	}{
		{"play_run without a target id is invalid", starter, "play_run", `{"play_id":"play-fix","target_type":"ticket"}`, apperrs.ErrInvalid},
		{"a ticket play on a doc is invalid", starter, "play_run", `{"play_id":"play-fix","target_type":"doc","target_id":"d-1"}`, apperrs.ErrInvalid},
		{"play_run of a missing play", starter, "play_run", `{"play_id":"missing","target_type":"ticket","target_id":"t-1"}`, apperrs.ErrNotFound},
		{"play_run without plays:run", "stranger", "play_run", `{"play_id":"play-fix","target_type":"ticket","target_id":"t-1"}`, apperrs.ErrForbidden},
		{"trail_list needs an id or a target", starter, "trail_list", `{"target_type":"ticket"}`, apperrs.ErrInvalid},
		{"trail_list of a missing trail", starter, "trail_list", `{"id":"missing"}`, apperrs.ErrNotFound},
		{"a stranger cannot read a trail", "stranger", "trail_list", `{"id":"tr-1"}`, apperrs.ErrForbidden},
		{"a stranger cannot list a target's trails", "stranger", "trail_list", `{"target_type":"ticket","target_id":"t-1"}`, apperrs.ErrForbidden},
		{"trail_update needs answer or stop", starter, "trail_update", `{"id":"tr-1"}`, apperrs.ErrInvalid},
		{"trail_update refuses answer and stop together", starter, "trail_update", `{"id":"tr-1","stop":true,"answer":{"q1":{"text":"x"}}}`, apperrs.ErrInvalid},
		{"an answer must be an object per question", starter, "trail_update", `{"id":"tr-1","answer":{"q1":"Yes"}}`, apperrs.ErrInvalid},
		{"stopping a missing trail", starter, "trail_update", `{"id":"missing","stop":true}`, apperrs.ErrNotFound},
		{"a stranger cannot stop someone's run", "stranger", "trail_update", `{"id":"tr-1","stop":true}`, apperrs.ErrForbidden},
		{"stopping a finished run conflicts", starter, "trail_update", `{"id":"tr-1","stop":true}`, apperrs.ErrConflict},
		{"answering a run that is not waiting conflicts", starter, "trail_update", `{"id":"tr-1","answer":{"q1":{"text":"x"}}}`, apperrs.ErrConflict},
		{"decisions_check_run without a ticket id is invalid", starter, "decisions_check_run", `{}`, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			seededTrail(f, "tr-1", TargetTicket, ticketID, fixedNow)
			_, err := callTool(t, RunMCPTools(f.runner), ctxAs(tt.actor), tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestTrailList_MissingTrailNamesTheListCall(t *testing.T) {
	f := newRunnerFixture()
	_, err := callTool(t, RunMCPTools(f.runner), ctxAs(starter), "trail_list", `{"id":"missing"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "trail_list with target_type and target_id")
}

func TestPlayRun_RefusalsNameTheRecoveryTools(t *testing.T) {
	f := newRunnerFixture()
	tools := RunMCPTools(f.runner)
	ctx := ctxAs(starter)

	_, err := callTool(t, tools, ctx, "play_run", `{"play_id":"missing","target_type":"ticket","target_id":"t-1"}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, err.Error(), "play_list shows the plays")

	run := `{"play_id":"play-fix","target_type":"ticket","target_id":"t-1"}`
	_, err = callTool(t, tools, ctx, "play_run", run)
	require.NoError(t, err)
	<-f.turns.done
	_, err = callTool(t, tools, ctx, "play_run", run)
	require.ErrorIs(t, err, apperrs.ErrConflict)
	assert.Contains(t, err.Error(), "trail_update with stop ends it")
}

func TestPlayRun_ThenTrailList(t *testing.T) {
	f := newRunnerFixture()
	tools := RunMCPTools(f.runner)
	ctx := ctxAs(starter)

	out, err := callTool(t, tools, ctx, "play_run", `{"play_id":"play-fix","target_type":"ticket","target_id":"t-1",`+
		`"memory_ids":["m-pick"],"custom_instructions":"go","move_to_status_id":"st-1"}`)
	require.NoError(t, err)
	<-f.turns.done
	run := out.(trailSummary)
	assert.Equal(t, ViaMCP, run.Via, "MCP runs carry their provenance")
	assert.Equal(t, TrailStarting, run.State)
	assert.Equal(t, "conv-ticket-"+ticketID, run.ConversationID)

	t.Run("a target's trails, newest first", func(t *testing.T) {
		out, err := callTool(t, tools, ctx, "trail_list", `{"target_type":"ticket","target_id":"t-1"}`)
		require.NoError(t, err)
		page := out.(mcptool.Page[trailSummary])
		require.Len(t, page.Items, 1)
		assert.Equal(t, run.ID, page.Items[0].ID)
	})
	t.Run("play_id filters them", func(t *testing.T) {
		out, err := callTool(t, tools, ctx, "trail_list", `{"target_type":"ticket","target_id":"t-1","play_id":"play-doc"}`)
		require.NoError(t, err)
		assert.Empty(t, out.(mcptool.Page[trailSummary]).Items)
	})
	t.Run("an id returns the trail with its run choices", func(t *testing.T) {
		out, err := callTool(t, tools, ctx, "trail_list", `{"id":"`+run.ID+`"}`)
		require.NoError(t, err)
		got := out.(trailDetail)
		assert.Equal(t, run.ID, got.ID)
		assert.Equal(t, []string{alwaysMem, pickedMem}, got.SelectedMemoryIDs)
		assert.Equal(t, "go", got.CustomInstructions)
		assert.Equal(t, "st-1", got.MoveToStatusID)
	})
}

func TestPlayRun_HarnessChoicePassesThrough(t *testing.T) {
	f := newRunnerFixture()
	f.harness.resolved = HarnessChoice{ComputerID: "c-resolved", Provider: "claude", Model: "sonnet-5"}

	out, err := callTool(t, RunMCPTools(f.runner), ctxAs(starter), "play_run", `{"play_id":"play-fix","target_type":"ticket",`+
		`"target_id":"t-1","computer_id":"c-picked","provider":"claude","model":"sonnet-5"}`)
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, HarnessChoice{ComputerID: "c-picked", Provider: "claude", Model: "sonnet-5"}, f.harness.lastChoice)
	got, err := f.runner.GetTrail(ctxAs(starter), out.(trailSummary).ID)
	require.NoError(t, err)
	assert.Equal(t, "c-resolved", got.ComputerID)
}

func TestTrailList_ReturnsTheNewestStepsWithoutRawDetail(t *testing.T) {
	f := newRunnerFixture()
	tr := seededTrail(f, "tr-long", TargetTicket, ticketID, fixedNow)
	for i := range 60 {
		tr.AppendActivity(ActivityEntry{Kind: harness.ActivityToolCall, Tool: "Bash", Summary: fmt.Sprintf("step %d", i),
			Detail: strings.Repeat("x", 100), At: fixedNow.Add(time.Duration(i) * time.Second)})
	}
	require.NoError(t, f.trails.UpdateTrail(t.Context(), tr))
	<-f.trails.terminal
	tools := RunMCPTools(f.runner)

	out, err := callTool(t, tools, ctxAs(starter), "trail_list", `{"id":"tr-long"}`)
	require.NoError(t, err)
	got := out.(trailDetail)
	assert.Equal(t, 60, got.StepsTotal)
	require.Len(t, got.Steps, defaultTrailSteps)
	assert.Equal(t, "step 10", got.Steps[0].Summary)
	assert.Equal(t, "step 59", got.Steps[len(got.Steps)-1].Summary)

	out, err = callTool(t, tools, ctxAs(starter), "trail_list", `{"id":"tr-long","steps":2}`)
	require.NoError(t, err)
	assert.Equal(t, []trailStep{
		{Kind: harness.ActivityToolCall, Tool: "Bash", Summary: "step 58", At: fixedNow.Add(58 * time.Second)},
		{Kind: harness.ActivityToolCall, Tool: "Bash", Summary: "step 59", At: fixedNow.Add(59 * time.Second)},
	}, out.(trailDetail).Steps)
}

func TestTrailUpdate_Stop(t *testing.T) {
	f := newRunnerFixture()
	tools := RunMCPTools(f.runner)
	out, err := callTool(t, tools, ctxAs(starter), "play_run", `{"play_id":"play-fix","target_type":"ticket","target_id":"t-1"}`)
	require.NoError(t, err)
	<-f.turns.done

	stopped, err := callTool(t, tools, ctxAs(starter), "trail_update", `{"id":"`+out.(trailSummary).ID+`","stop":true}`)
	require.NoError(t, err)
	assert.Equal(t, TrailInterrupted, stopped.(trailSummary).State)
}

func TestTrailUpdate_Answer(t *testing.T) {
	f := heldFixture(t)
	tools := RunMCPTools(f.runner)
	out, err := callTool(t, tools, ctxAs(starter), "play_run", `{"play_id":"play-fix","target_type":"ticket","target_id":"t-1"}`)
	require.NoError(t, err)
	<-f.turns.done
	id := out.(trailSummary).ID
	obs := f.turns.last().Observer
	obs.OnStarted("sess-1")
	obs.OnQuestion(harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{ID: "q1", Text: "Which?", MultiSelect: true}, {ID: "q2", Text: "Name?"}}})

	waiting, err := callTool(t, tools, ctxAs(starter), "trail_list", `{"id":"`+id+`"}`)
	require.NoError(t, err)
	require.NotNil(t, waiting.(trailDetail).Question, "the open question and its ids are readable before answering")
	assert.Equal(t, "q1", waiting.(trailDetail).Question.Questions[0].ID)

	answered, err := callTool(t, tools, ctxAs(starter), "trail_update",
		`{"id":"`+id+`","answer":{"q1":{"selected":["A","B"]},"q2":{"text":"Bot"}}}`)
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, answered.(trailSummary).State)
	require.Len(t, f.turns.answered, 1)
	assert.Equal(t, harness.AnswerValue{Selected: []string{"A", "B"}}, f.turns.answered[0].Answers["q1"])
	assert.Equal(t, harness.AnswerValue{Text: "Bot"}, f.turns.answered[0].Answers["q2"])
}
