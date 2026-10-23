package plays

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func TestPlayUpdate_AddsUpdatesAndRemovesAutoPlays(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	tools := MCPTools(s)
	play := ps[TypeTicket]

	out, err := callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","add_auto_plays":[
		{"moment":"ticket.unblocked","conditions":{"match":"all","groups":[{"match":"any","rules":[{"field":"type","op":"is","values":["Bug"]},{"field":"blocked","op":"unset"}]}]},
		 "priority":{"rules":[{"level":"high","when":{"match":"all","rules":[{"field":"label","op":"is","values":["urgent"]}]}}],"otherwise":"low"},"once_within_minutes":60},
		{"moment":"ticket.entered_stage","moment_stage":"done","run_on":"causer"}]}`)
	require.NoError(t, err)
	added := out.(playResult).AutoPlays
	require.Len(t, added, 2)
	assert.False(t, added[0].Enabled)
	assert.Equal(t, LevelLow, added[0].Priority.Otherwise)
	assert.Equal(t, RunOnDeveloper, added[0].RunOn)
	assert.Equal(t, "Fix with AI", out.(playResult).Label, "no play field was passed, so the play is untouched")

	out, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`",
		"update_auto_plays":[{"id":"`+added[0].ID+`","enabled":true}],"remove_auto_plays":["`+added[1].ID+`"]}`)
	require.NoError(t, err)
	left := out.(playResult).AutoPlays
	require.Len(t, left, 1)
	assert.True(t, left[0].Enabled)
	assert.Equal(t, 60, left[0].OnceWithinMinutes, "fields the change left out keep their values")
	assert.Equal(t, added[0].Conditions, left[0].Conditions)

	var topics []string
	for _, e := range repo.published {
		topics = append(topics, e.Topic)
	}
	assert.Equal(t, []string{TopicAutoPlayCreated, TopicAutoPlayCreated, TopicAutoPlayUpdated, TopicAutoPlayDeleted}, topics,
		"no play.updated: only auto plays changed")
}

func TestPlayUpdate_AutoPlayFailuresSayWhatTookEffect(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	tools := MCPTools(s)
	play := ps[TypeTicket]

	_, err := callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","add_auto_plays":[{"moment":"doc.changed"}]}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	var partial *mcptool.PartialError
	assert.False(t, errors.As(err, &partial), "nothing took effect, so there is nothing to report")

	_, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`",
		"add_auto_plays":[{"moment":"ticket.created"}],"update_auto_plays":[{"id":"ghost","enabled":true}]}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	require.ErrorAs(t, err, &partial)
	require.Len(t, partial.Applied, 1)
	assert.Contains(t, partial.Applied[0], "added auto play ")
	assert.Len(t, repo.autoPlays, 1, "the add before the failure stays")

	_, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","label":"Fix it","add_auto_plays":[{"moment":"ticket.created"}]}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "play fields still need plays:write")

	_, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","remove_auto_plays":["ghost"]}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestPlayList_AttachesAutoPlaysInOneReadForTheirReaders(t *testing.T) {
	s, repo, ps := autoPlayFixture(t)
	tools := MCPTools(s)
	_, err := s.CreateAutoPlay(ctxAs("editor"), workspaceID, ps[TypeTicket].ID, AutoPlayInput{Moment: MomentTicketCreated})
	require.NoError(t, err)

	out, err := callTool(t, tools, ctxAs("editor"), "play_list", `{"workspace_id":"$WS"}`)
	require.NoError(t, err)
	byLabel := map[string][]autoPlayResult{}
	for _, p := range out.(mcptool.Page[playResult]).Items {
		byLabel[p.Label] = p.AutoPlays
	}
	assert.Len(t, byLabel["Fix with AI"], 1)
	assert.Equal(t, []autoPlayResult{}, byLabel["To tickets"], "a reader sees an empty list, not a missing one")
	assert.Equal(t, 1, repo.autoPlayLists, "one batch read for the whole page")

	out, err = callTool(t, tools, ctxAs("player"), "play_list", `{"workspace_id":"$WS"}`)
	require.NoError(t, err)
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "auto_plays", "left out without autoplays:read")

	repo.autoPlayErr = assert.AnError
	_, err = callTool(t, tools, ctxAs("editor"), "play_list", `{"workspace_id":"$WS"}`)
	require.ErrorIs(t, err, assert.AnError)
}

func TestPlayCreateAndUpdate_StackPlayRunsOnAChosenComputer(t *testing.T) {
	s, repo, _ := computerFixture(t)
	tools := MCPTools(s)

	out, err := callTool(t, tools, ctxAs("owner"), "play_create", `{"workspace_id":"$WS","label":"Look into it","type":"stack","instructions":"Look."}`)
	require.NoError(t, err)
	play := out.(playResult)
	assert.Equal(t, TypeStack, play.Type)

	out, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","add_auto_plays":[
		{"moment":"stack.service_crashed","computer_id":"c-run","conditions":{"match":"all","groups":[{"match":"all","rules":[{"field":"service","op":"is","values":["api"]}]}]}}]}`)
	require.NoError(t, err)
	added := out.(playResult).AutoPlays
	require.Len(t, added, 1)
	assert.Equal(t, []any{false, MomentStackServiceCrashed, RunOnComputer, "c-run"}, []any{added[0].Enabled, added[0].Moment, added[0].RunOn, added[0].ComputerID},
		"a stack play's auto play runs on the computer, with no run_on to say")

	out, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","update_auto_plays":[{"id":"`+added[0].ID+`","enabled":true,"computer_id":"c-own"}]}`)
	require.NoError(t, err)
	assert.Equal(t, []any{true, "c-own"}, []any{out.(playResult).AutoPlays[0].Enabled, out.(playResult).AutoPlays[0].ComputerID})
	assert.Equal(t, "editor", repo.autoPlays[0].RunAs, "the run takes the identity of whoever picked the computer")

	_, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+play.ID+`","update_auto_plays":[{"id":"`+added[0].ID+`","computer_id":"c-see"}]}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a computer shared for seeing only cannot be picked")
}

func TestPlayUpdate_AutoPlayComputersAreAnOrderedListTheModelCanKeepOrReplace(t *testing.T) {
	s, repo, _ := computerFixture(t)
	tools := MCPTools(s)
	out, err := callTool(t, tools, ctxAs("owner"), "play_create", `{"workspace_id":"$WS","label":"Look into it","type":"stack","instructions":"Look."}`)
	require.NoError(t, err)
	playID := out.(playResult).ID
	update := func(changes string) []autoPlayResult {
		t.Helper()
		out, err := callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+playID+`",`+changes+`}`)
		require.NoError(t, err)
		return out.(playResult).AutoPlays
	}

	added := update(`"add_auto_plays":[{"moment":"stack.service_crashed","computer_ids":["c-run","c-own"]}]`)
	assert.Equal(t, []any{[]string{"c-run", "c-own"}, "c-run", "editor"}, []any{added[0].ComputerIDs, added[0].ComputerID, added[0].RunAs}, "in the order the model gave")
	id := added[0].ID

	kept := update(`"update_auto_plays":[{"id":"` + id + `","enabled":true}]`)
	assert.Equal(t, []any{true, []string{"c-run", "c-own"}}, []any{kept[0].Enabled, kept[0].ComputerIDs}, "omitting computer_ids keeps the list")

	reordered := update(`"update_auto_plays":[{"id":"` + id + `","computer_ids":["c-own","c-run"]}]`)
	assert.Equal(t, []string{"c-own", "c-run"}, reordered[0].ComputerIDs)

	yours := update(`"update_auto_plays":[{"id":"` + id + `","computer_ids":[]}]`)
	assert.Equal(t, []any{[]string{}, "editor"}, []any{yours[0].ComputerIDs, yours[0].RunAs}, "an empty list is the caller's own computers in their order")
	assert.Equal(t, []string{}, repo.autoPlays[0].ComputerIDs)

	_, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"`+playID+`","update_auto_plays":[{"id":"`+id+`","computer_ids":["c-own","c-see"]}]}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "one computer in the list the caller may not run agents on refuses it")
}
