package plays

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHandler_Run_Accepted(t *testing.T) {
	f := newRunnerFixture()
	h := NewRunHandler(f.runner).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run",
		`{"target_type":"ticket","target_id":"t-1","memory_ids":["m-pick"],"custom_instructions":"go","move_to_status_id":"st-1"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done

	var tr Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tr))
	assert.Equal(t, TrailStarting, tr.State)
	assert.Equal(t, ViaWeb, tr.Via)
	assert.Equal(t, "st-1", tr.MoveToStatusID)
	assert.Equal(t, []string{alwaysMem, pickedMem}, tr.SelectedMemoryIDs)
}

func TestRunHandler_Run_HarnessChoicePassesThrough(t *testing.T) {
	f := newRunnerFixture()
	f.harness.resolved = HarnessChoice{ComputerID: "c-resolved", Provider: "claude", Model: "sonnet-5"}
	h := NewRunHandler(f.runner).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run",
		`{"target_type":"ticket","target_id":"t-1","computer_id":"c-picked","provider":"claude","model":"sonnet-5"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done

	assert.Equal(t, HarnessChoice{ComputerID: "c-picked", Provider: "claude", Model: "sonnet-5"}, f.harness.lastChoice)
	var tr Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tr))
	assert.Equal(t, "c-resolved", tr.ComputerID)
	assert.Equal(t, "claude", tr.Provider)
	assert.Equal(t, "sonnet-5", tr.Model)
}

func TestRunHandler_Run_Errors(t *testing.T) {
	tests := []struct {
		name string
		body string
		user string
		want int
	}{
		{"malformed body", `{`, starter, http.StatusBadRequest},
		{"no actor", `{"target_type":"ticket","target_id":"t-1"}`, "", http.StatusUnauthorized},
		{"wrong stage", `{"target_type":"ticket","target_id":"t-backlog"}`, starter, http.StatusBadRequest},
		{"forbidden", `{"target_type":"ticket","target_id":"t-1"}`, "stranger", http.StatusForbidden},
		{"unknown ticket", `{"target_type":"ticket","target_id":"nope"}`, starter, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunnerFixture()
			rec := do(t, NewRunHandler(f.runner).Routes(), http.MethodPost, "/api/plays/"+fixPlayID+"/run", tt.body, tt.user)
			assert.Equal(t, tt.want, rec.Code, rec.Body.String())
		})
	}
}

func TestRunHandler_Stop(t *testing.T) {
	f := newRunnerFixture()
	h := NewRunHandler(f.runner).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run", `{"target_type":"ticket","target_id":"t-1"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done
	var tr Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tr))

	assert.Equal(t, http.StatusForbidden, do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/stop", "", "stranger").Code)
	assert.Equal(t, http.StatusNotFound, do(t, h, http.MethodPost, "/api/plays/runs/missing/stop", "", starter).Code)

	rec = do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/stop", "", starter)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var stopped Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stopped))
	assert.Equal(t, TrailInterrupted, stopped.State)

	assert.Equal(t, http.StatusConflict, do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/stop", "", starter).Code)
}

func TestRunHandler_Run_SecondRun_Conflicts(t *testing.T) {
	f := newRunnerFixture()
	h := NewRunHandler(f.runner).Routes()
	body := `{"target_type":"ticket","target_id":"t-1"}`
	require.Equal(t, http.StatusAccepted, do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run", body, starter).Code)
	<-f.turns.done
	rec := do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run", body, starter)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "a run is in progress")
}

func TestRunHandler_GetAndList(t *testing.T) {
	f := newRunnerFixture()
	seededTrail(f, "tr-1", TargetTicket, ticketID, fixedNow)
	h := NewRunHandler(f.runner).Routes()

	rec := do(t, h, http.MethodGet, "/api/plays/runs/tr-1", "", starter)
	require.Equal(t, http.StatusOK, rec.Code)
	var tr Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tr))
	assert.Equal(t, "tr-1", tr.ID)

	assert.Equal(t, http.StatusNotFound, do(t, h, http.MethodGet, "/api/plays/runs/missing", "", starter).Code)
	assert.Equal(t, http.StatusForbidden, do(t, h, http.MethodGet, "/api/plays/runs/tr-1", "", "stranger").Code)

	rec = do(t, h, http.MethodGet, "/api/plays/runs?target_type=ticket&target_id="+ticketID, "", starter)
	require.Equal(t, http.StatusOK, rec.Code)
	var list []Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Len(t, list, 1)

	assert.Equal(t, http.StatusBadRequest, do(t, h, http.MethodGet, "/api/plays/runs?target_type=column&target_id=x", "", starter).Code)
}

func TestRunHandler_LatestChoices(t *testing.T) {
	f := newRunnerFixture()
	seededTrail(f, "tr-1", TargetTicket, ticketID, fixedNow)
	h := NewRunHandler(f.runner).Routes()

	rec := do(t, h, http.MethodGet, "/api/plays/latest-choices?play_id="+fixPlayID+"&project_id="+projectID, "", starter)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var c Choices
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &c))
	assert.Equal(t, Choices{MemoryIDs: []string{alwaysMem}, MoveToStatusID: "st-tr-1"}, c)

	assert.Equal(t, http.StatusBadRequest, do(t, h, http.MethodGet, "/api/plays/latest-choices?play_id="+fixPlayID, "", starter).Code)
	assert.Equal(t, http.StatusForbidden, do(t, h, http.MethodGet, "/api/plays/latest-choices?play_id="+fixPlayID+"&project_id="+projectID, "", "stranger").Code)
}

func TestRunHandler_Active(t *testing.T) {
	f := newRunnerFixture()
	h := NewRunHandler(f.runner).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run", `{"target_type":"ticket","target_id":"t-1"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done
	var tr Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tr))

	rec = do(t, h, http.MethodGet, "/api/plays/runs/active?ticket_ids=t-1,t-2,", "", starter)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		Active map[string]string `json:"active"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, map[string]string{"t-1": tr.ID}, body.Active)

	rec = do(t, h, http.MethodGet, "/api/plays/runs/active?ticket_ids=t-1", "", "stranger")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"active":{}}`, rec.Body.String(), "a caller without plays:read sees no trail rather than an error")

	rec = do(t, h, http.MethodGet, "/api/plays/runs/active", "", starter)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"active":{}}`, rec.Body.String())
}

func TestRunHandler_Answer(t *testing.T) {
	f := heldFixture(t)
	h := NewRunHandler(f.runner).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/"+fixPlayID+"/run", `{"target_type":"ticket","target_id":"t-1"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done
	var tr Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tr))
	obs := f.turns.last().Observer
	obs.OnStarted("sess-1")

	body := `{"answers":{"q1":{"selected":["Yes"]}}}`
	assert.Equal(t, http.StatusConflict, do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/answer", body, starter).Code, "nothing asked yet")
	obs.OnQuestion(askedQuestion())
	assert.Equal(t, http.StatusForbidden, do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/answer", body, "stranger").Code)
	assert.Equal(t, http.StatusNotFound, do(t, h, http.MethodPost, "/api/plays/runs/missing/answer", body, starter).Code)
	assert.Equal(t, http.StatusBadRequest, do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/answer", `{`, starter).Code)

	rec = do(t, h, http.MethodPost, "/api/plays/runs/"+tr.ID+"/answer", body, starter)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var answered Trail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &answered))
	assert.Equal(t, TrailRunning, answered.State)
	require.NotNil(t, answered.Question)
	assert.Equal(t, []string{"Yes"}, answered.Question.Answer.Answers["q1"].Selected)
}
