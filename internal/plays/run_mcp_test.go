package plays

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

func TestRunMCPTools_Shape(t *testing.T) {
	tools := RunMCPTools(newRunnerFixture().runner)
	var names []string
	for _, tool := range tools {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Description)
		assert.NotNil(t, tool.InputSchema)
		assert.NotNil(t, tool.Call)
	}
	assert.ElementsMatch(t, []string{"play_run", "play_run_get", "play_run_stop", "play_run_answer", "decisions_check_run", "play_list_runs"}, names)
}

func TestRunMCPTools_Stop(t *testing.T) {
	f := newRunnerFixture()
	tools := RunMCPTools(f.runner)
	out, err := mcpToolByName(t, tools, "play_run").Call(ctxAs(starter), map[string]any{"play_id": fixPlayID, "target_type": "ticket", "target_id": ticketID})
	require.NoError(t, err)
	<-f.turns.done
	trail := out.(*Trail)

	_, err = mcpToolByName(t, tools, "play_run_stop").Call(ctxAs("stranger"), map[string]any{"id": trail.ID})
	assert.ErrorIs(t, err, apperrs.ErrForbidden)

	stopped, err := mcpToolByName(t, tools, "play_run_stop").Call(ctxAs(starter), map[string]any{"id": trail.ID})
	require.NoError(t, err)
	assert.Equal(t, TrailInterrupted, stopped.(*Trail).State)
}

func TestRunMCPTools_RunGetList(t *testing.T) {
	f := newRunnerFixture()
	tools := RunMCPTools(f.runner)
	ctx := ctxAs(starter)

	out, err := mcpToolByName(t, tools, "play_run").Call(ctx, map[string]any{
		"play_id": fixPlayID, "target_type": "ticket", "target_id": ticketID,
		"memory_ids": []any{pickedMem}, "custom_instructions": "go", "move_to_status_id": "st-1",
	})
	require.NoError(t, err)
	<-f.turns.done
	trail := out.(*Trail)
	assert.Equal(t, ViaMCP, trail.Via, "MCP runs carry their provenance")
	assert.Equal(t, TrailStarting, trail.State)
	assert.Equal(t, []string{alwaysMem, pickedMem}, trail.SelectedMemoryIDs)
	assert.Equal(t, "st-1", trail.MoveToStatusID)

	got, err := mcpToolByName(t, tools, "play_run_get").Call(ctx, map[string]any{"id": trail.ID})
	require.NoError(t, err)
	assert.Equal(t, trail.ID, got.(*Trail).ID)

	list, err := mcpToolByName(t, tools, "play_list_runs").Call(ctx, map[string]any{"target_type": "ticket", "target_id": ticketID})
	require.NoError(t, err)
	assert.Len(t, list.([]*Trail), 1)
}

func TestRunMCPTools_Run_HarnessChoicePassesThrough(t *testing.T) {
	f := newRunnerFixture()
	f.harness.resolved = HarnessChoice{ComputerID: "c-resolved", Provider: "claude", Model: "sonnet-5"}
	tools := RunMCPTools(f.runner)

	out, err := mcpToolByName(t, tools, "play_run").Call(ctxAs(starter), map[string]any{
		"play_id": fixPlayID, "target_type": "ticket", "target_id": ticketID,
		"computer_id": "c-picked", "provider": "claude", "model": "sonnet-5",
	})
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, HarnessChoice{ComputerID: "c-picked", Provider: "claude", Model: "sonnet-5"}, f.harness.lastChoice)
	trail := out.(*Trail)
	assert.Equal(t, "c-resolved", trail.ComputerID)
	assert.Equal(t, "claude", trail.Provider)
	assert.Equal(t, "sonnet-5", trail.Model)
}

func TestRunMCPTools_MissingArgs_AreInvalid(t *testing.T) {
	tools := RunMCPTools(newRunnerFixture().runner)
	ctx := ctxAs(starter)
	_, err := mcpToolByName(t, tools, "play_run").Call(ctx, map[string]any{"play_id": fixPlayID})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = mcpToolByName(t, tools, "play_run_get").Call(ctx, map[string]any{})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = mcpToolByName(t, tools, "play_run_stop").Call(ctx, map[string]any{})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = mcpToolByName(t, tools, "play_list_runs").Call(ctx, map[string]any{"target_type": "ticket"})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestRunMCPTools_Answer(t *testing.T) {
	f := heldFixture(t)
	tools := RunMCPTools(f.runner)
	out, err := mcpToolByName(t, tools, "play_run").Call(ctxAs(starter), map[string]any{"play_id": fixPlayID, "target_type": "ticket", "target_id": ticketID})
	require.NoError(t, err)
	<-f.turns.done
	trail := out.(*Trail)
	obs := f.turns.last().Observer
	obs.OnStarted("sess-1")
	obs.OnQuestion(harness.Question{RequestID: "req-1", Questions: []harness.QuestionItem{{ID: "q1", Text: "Which?", MultiSelect: true}, {ID: "q2", Text: "Name?"}}})

	_, err = mcpToolByName(t, tools, "play_run_answer").Call(ctxAs(starter), map[string]any{"id": trail.ID})
	assert.ErrorIs(t, err, apperrs.ErrInvalid, "answers are required")

	answered, err := mcpToolByName(t, tools, "play_run_answer").Call(ctxAs(starter), map[string]any{
		"id": trail.ID, "answers": map[string]any{"q1": []any{"A", "B"}, "q2": "Bot"},
	})
	require.NoError(t, err)
	assert.Equal(t, TrailRunning, answered.(*Trail).State)
	require.Len(t, f.turns.answered, 1)
	assert.Equal(t, harness.AnswerValue{Selected: []string{"A", "B"}}, f.turns.answered[0].Answers["q1"], "an array is a multi-select")
	assert.Equal(t, harness.AnswerValue{Text: "Bot"}, f.turns.answered[0].Answers["q2"], "a string is the typed or picked answer")
}
