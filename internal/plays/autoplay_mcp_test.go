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

	_, err = callTool(t, tools, ctxAs("editor"), "play_update", `{"workspace_id":"$WS","id":"decisions-check","enabled":true,"remove_auto_plays":["x"]}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "the decisions check takes only enabled")
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
