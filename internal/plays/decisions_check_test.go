package plays

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

const checkPlayID = "play-decisions"

// newDecisionsFixture is the runner fixture with a progress column, two done columns, a done ticket, and the seeded
// decisions check play.
func newDecisionsFixture() *runnerFixture {
	f := newRunnerFixture()
	f.targets.statuses["st-progress"] = StatusTarget{Name: "Doing", Stage: StageProgress}
	f.targets.statuses["st-shipped"] = StatusTarget{Name: "Shipped", Stage: StageDone}
	f.targets.setTicketStage(ticketID, StageDone)
	done := StageDone
	f.plays.byID[checkPlayID] = &Play{
		ID: checkPlayID, WorkspaceID: workspaceID, Label: decisionsCheckLabel, Type: TypeTicket,
		Instructions: decisionsCheckInstructions, Enabled: true, ShowWhenStage: &done, BuiltinKey: DecisionsCheckKey,
	}
	return f
}

func TestRetryDecisionsCheck_RunsTheSeededPlayOnTheCallersHarness(t *testing.T) {
	f := newDecisionsFixture()

	trail, err := f.runner.RetryDecisionsCheck(ctxAs(starter), ticketID, ViaWeb)
	require.NoError(t, err)
	<-f.turns.done

	assert.Equal(t, []any{checkPlayID, starter, TrailStarting}, []any{trail.PlayID, trail.StarterID, trail.State})
	req := f.turns.last()
	assert.Equal(t, starter, req.ViaUserID)
	require.NotNil(t, req.Play)
	assert.Contains(t, req.Play.Instructions, "decisions_log")
	assert.Equal(t, "Started Decisions check", f.threads.snapshot()[0].body)
}

func TestRetryDecisionsCheck_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		ticket  string
		arrange func(f *runnerFixture)
		want    error
	}{
		{"no actor", context.Background(), ticketID, nil, apperrs.ErrUnauthorized},
		{"no ticket id", ctxAs(starter), " ", nil, apperrs.ErrInvalid},
		{"unknown ticket", ctxAs(starter), "t-missing", nil, apperrs.ErrNotFound},
		{"ticket not done", ctxAs(starter), ticketID, func(f *runnerFixture) { f.targets.setTicketStage(ticketID, StageTesting) }, apperrs.ErrInvalid},
		{"caller may not run plays", ctxAs("u-stranger"), ticketID, nil, apperrs.ErrForbidden},
		{"the play was deleted", ctxAs(starter), ticketID, func(f *runnerFixture) { delete(f.plays.byID, checkPlayID) }, apperrs.ErrNotFound},
		{"the play is disabled", ctxAs(starter), ticketID, func(f *runnerFixture) { f.plays.byID[checkPlayID].Enabled = false }, apperrs.ErrInvalid},
		{"the plays cannot be read", ctxAs(starter), ticketID, func(f *runnerFixture) { f.plays.listErr = errors.New("disk") }, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDecisionsFixture()
			if tt.arrange != nil {
				tt.arrange(f)
			}
			_, err := f.runner.RetryDecisionsCheck(tt.ctx, tt.ticket, ViaWeb)
			require.Error(t, err)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
			}
			assert.Empty(t, f.trails.all(), "a refused retry leaves no trail")
		})
	}
}

func TestDecisionsCheckRun_HTTPAndMCP(t *testing.T) {
	f := newDecisionsFixture()
	h := NewRunHandler(f.runner, nil).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/decisions-check", `{"ticket_id":"`+ticketID+`"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done
	assert.Equal(t, http.StatusBadRequest, do(t, h, http.MethodPost, "/api/plays/decisions-check", `{`, starter).Code)
	assert.Equal(t, http.StatusForbidden, do(t, h, http.MethodPost, "/api/plays/decisions-check", `{"ticket_id":"`+ticketID+`"}`, "u-stranger").Code)

	f2 := newDecisionsFixture()
	tools := RunMCPTools(f2.runner)
	out, err := callTool(t, tools, ctxAs(starter), "play_run", `{"decisions_check":true,"target_type":"ticket","target_id":"`+ticketID+`"}`)
	require.NoError(t, err)
	<-f2.turns.done
	assert.Equal(t, ViaMCP, out.(trailSummary).Via)
	assert.Equal(t, checkPlayID, out.(trailSummary).PlayID)
	_, err = callTool(t, tools, ctxAs(starter), "play_run", `{"decisions_check":true,"target_type":"ticket","target_id":""}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestDecisionsCheckSwitch_IsGone_TheSeededPlayListsInstead(t *testing.T) {
	repo := newFakeRepo()
	perm := newFakePerm(map[string][]permissions.Action{
		"owner": {permissions.AutomationsWrite, permissions.PlaysRead, permissions.PlaysWrite, permissions.AutoplaysRead},
	})
	s := newTestService(repo, perm)
	require.NoError(t, s.SeedDefaults(context.Background(), workspaceID))
	h := NewHandler(s).Routes()
	path := "/api/workspaces/" + workspaceID + "/plays/decisions-check"

	assert.Equal(t, http.StatusNotFound, do(t, h, http.MethodGet, path, "", "owner").Code)
	assert.Equal(t, http.StatusNotFound, do(t, h, http.MethodPatch, path, `{"enabled":true}`, "owner").Code)

	tools := MCPTools(s)
	_, err := callTool(t, tools, ctxAs("owner"), "play_update", `{"workspace_id":"`+workspaceID+`","id":"decisions-check","enabled":true}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)

	listed, err := callTool(t, tools, ctxAs("owner"), "play_list", `{"workspace_id":"`+workspaceID+`"}`)
	require.NoError(t, err)
	raw, err := json.Marshal(listed)
	require.NoError(t, err)
	var page struct {
		Items []playResult `json:"items"`
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	var check *playResult
	for i, p := range page.Items {
		assert.NotEqual(t, DecisionsCheckKey, p.ID, "no pseudo play rides on the list")
		if p.Label == decisionsCheckLabel {
			check = &page.Items[i]
		}
	}
	require.NotNil(t, check)
	require.Len(t, check.AutoPlays, 1)
	assert.False(t, check.AutoPlays[0].Enabled, "a new workspace starts with the check off")
}
