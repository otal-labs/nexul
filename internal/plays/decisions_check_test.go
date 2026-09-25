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
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// newDecisionsFixture is the runner fixture with a progress column, two done columns, and a user who may not run plays.
func newDecisionsFixture() *runnerFixture {
	f := newRunnerFixture()
	f.targets.statuses["st-progress"] = StatusTarget{Name: "Doing", Stage: StageProgress}
	f.targets.statuses["st-shipped"] = StatusTarget{Name: "Shipped", Stage: StageDone}
	f.targets.setTicketStage(ticketID, StageDone)
	return f
}

func moveEvent(t *testing.T, from, to, actorKind, userID, developer string) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"ticket": map[string]any{"id": ticketID, "developer": developer},
		"from":   from, "to": to,
		"actor": map[string]any{"kind": actorKind, "user_id": userID},
	})
	require.NoError(t, err)
	return eventbus.Event{ID: "ev-1", Topic: "ticket.status_changed", Payload: raw}
}

func decisionTrails(f *runnerFixture) []*Trail {
	var out []*Trail
	for _, t := range f.trails.all() {
		if t.PlayID == DecisionsCheckPlayID {
			out = append(out, t)
		}
	}
	return out
}

func TestHandleTicketStatusChanged_EnteringDone_FiresExactlyOneCheck(t *testing.T) {
	f := newDecisionsFixture()
	ev := moveEvent(t, "st-progress", "st-done", "user", starter, "")

	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), ev))
	<-f.turns.done
	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), ev), "a redelivery is a no-op")
	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), moveEvent(t, "st-progress", "st-shipped", "user", starter, "")))

	trails := decisionTrails(f)
	require.Len(t, trails, 1)
	assert.Equal(t, starter, trails[0].StarterID, "runs on the mover's own harness")
	assert.Equal(t, decisionsCheckLabel, trails[0].PlayLabel)
	assert.Equal(t, ViaWeb, trails[0].Via)
	req := f.turns.last()
	assert.Equal(t, starter, req.ViaUserID)
	assert.Contains(t, req.ExtraRequestBlocks[0], "Play: Decisions check")
	assert.Contains(t, req.ExtraRequestBlocks[0], "decisions_log")
	assert.Equal(t, "Started Decisions check", f.threads.snapshot()[0].body)
}

func TestHandleTicketStatusChanged_NotAnEntryIntoDone_DoesNothing(t *testing.T) {
	tests := []struct{ name, from, to string }{
		{"into a progress column", "st-review", "st-progress"},
		{"between two done columns", "st-done", "st-shipped"},
		{"into a deleted column", "st-progress", "st-gone"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDecisionsFixture()
			require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), moveEvent(t, tt.from, tt.to, "user", starter, "")))
			assert.Empty(t, f.trails.all())
		})
	}
}

func TestHandleTicketStatusChanged_FromAMissingColumn_CountsAsEntering(t *testing.T) {
	f := newDecisionsFixture()
	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), moveEvent(t, "st-gone", "st-done", "user", starter, "")))
	<-f.turns.done
	assert.Len(t, decisionTrails(f), 1)
}

func TestHandleTicketStatusChanged_MergedPRMove_RunsOnTheDeveloper(t *testing.T) {
	f := newDecisionsFixture()
	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), moveEvent(t, "", "st-done", "automation", "", "login-"+starter)))
	<-f.turns.done
	trails := decisionTrails(f)
	require.Len(t, trails, 1)
	assert.Equal(t, starter, trails[0].StarterID)
}

func TestHandleTicketStatusChanged_McpMove_CarriesMcpProvenance(t *testing.T) {
	f := newDecisionsFixture()
	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), moveEvent(t, "st-progress", "st-done", "play:mcp", starter, "")))
	<-f.turns.done
	assert.Equal(t, ViaMCP, decisionTrails(f)[0].Via)
}

func TestHandleTicketStatusChanged_CannotStart_KeepsAFailedTrailOnTheTicket(t *testing.T) {
	tests := []struct {
		name      string
		arrange   func(f *runnerFixture)
		ev        func(t *testing.T) eventbus.Event
		wantError string
		wantEvent bool
	}{
		{
			name:      "no developer to run it for",
			ev:        func(t *testing.T) eventbus.Event { return moveEvent(t, "st-progress", "st-done", "automation", "", "") },
			wantError: "nobody to run it for",
		},
		{
			name: "developer is not a known user",
			ev: func(t *testing.T) eventbus.Event {
				return moveEvent(t, "st-progress", "st-done", "automation", "", "ghost")
			},
			wantError: "nobody to run it for",
		},
		{
			name: "mover may not run plays",
			ev: func(t *testing.T) eventbus.Event {
				return moveEvent(t, "st-progress", "st-done", "user", "u-stranger", "")
			},
			wantError: "plays:run",
			wantEvent: true,
		},
		{
			name:      "setup gate refuses the provider",
			arrange:   func(f *runnerFixture) { f.harness.err = errors.New("the provider is not confirmed on this computer") },
			ev:        func(t *testing.T) eventbus.Event { return moveEvent(t, "st-progress", "st-done", "user", starter, "") },
			wantError: "not confirmed",
			wantEvent: true,
		},
		{
			name: "another run holds the ticket",
			arrange: func(f *runnerFixture) {
				require.NoError(t, f.trails.CreateTrail(context.Background(), &Trail{ID: "busy", PlayID: fixPlayID, TargetType: TargetTicket, TargetID: ticketID, State: TrailRunning}))
			},
			ev:        func(t *testing.T) eventbus.Event { return moveEvent(t, "st-progress", "st-done", "user", starter, "") },
			wantError: "in progress",
			wantEvent: true,
		},
		{
			name:      "the thread cannot be opened",
			arrange:   func(f *runnerFixture) { f.threads.ticketErr = errors.New("chat down") },
			ev:        func(t *testing.T) eventbus.Event { return moveEvent(t, "st-progress", "st-done", "user", starter, "") },
			wantError: "chat down",
			wantEvent: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDecisionsFixture()
			if tt.arrange != nil {
				tt.arrange(f)
			}
			require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), tt.ev(t)))
			trails := decisionTrails(f)
			require.Len(t, trails, 1)
			assert.Equal(t, TrailFailed, trails[0].State)
			assert.Contains(t, trails[0].LastError, tt.wantError)
			assert.Equal(t, tt.wantEvent, len(f.trails.eventsFor(TopicRunFinished)) == 1)
			frames := f.live.snapshot()
			require.NotEmpty(t, frames, "the ticket page learns of it live")
			assert.Equal(t, TrailFailed, frames[len(frames)-1].State)
		})
	}
}

func TestHandleTicketStatusChanged_Errors(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(f *runnerFixture)
		ev      func(t *testing.T) eventbus.Event
		want    error
	}{
		{"bad payload", nil, func(*testing.T) eventbus.Event { return eventbus.Event{Payload: []byte("{")} }, apperrs.ErrFatal},
		{"no ticket id", nil, func(*testing.T) eventbus.Event { return eventbus.Event{Payload: []byte(`{"to":"st-done"}`)} }, apperrs.ErrFatal},
		{"trails unreadable", func(f *runnerFixture) { f.trails.listErr = errors.New("disk") }, func(t *testing.T) eventbus.Event {
			return moveEvent(t, "st-progress", "st-done", "user", starter, "")
		}, apperrs.ErrRetryable},
		{"developer lookup fails", nil, func(t *testing.T) eventbus.Event {
			return moveEvent(t, "st-progress", "st-done", "automation", "", "broken")
		}, apperrs.ErrRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDecisionsFixture()
			if tt.arrange != nil {
				tt.arrange(f)
			}
			err := f.runner.HandleTicketStatusChanged(context.Background(), tt.ev(t))
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestRetryDecisionsCheck_StartsOnTheCallersHarness(t *testing.T) {
	f := newDecisionsFixture()
	require.NoError(t, f.runner.HandleTicketStatusChanged(context.Background(), moveEvent(t, "st-progress", "st-done", "automation", "", "")))
	require.Equal(t, TrailFailed, decisionTrails(f)[0].State)

	trail, err := f.runner.RetryDecisionsCheck(ctxAs(starter), ticketID, ViaWeb)
	require.NoError(t, err)
	<-f.turns.done
	assert.Equal(t, TrailStarting, trail.State)
	assert.Equal(t, starter, trail.StarterID)
	assert.Len(t, decisionTrails(f), 2)
}

func TestRetryDecisionsCheck_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		ticket  string
		stage   Stage
		want    error
		records bool
	}{
		{"no actor", context.Background(), ticketID, StageDone, apperrs.ErrUnauthorized, false},
		{"no ticket id", ctxAs(starter), " ", StageDone, apperrs.ErrInvalid, false},
		{"unknown ticket", ctxAs(starter), "t-missing", StageDone, apperrs.ErrNotFound, false},
		{"ticket not done", ctxAs(starter), ticketID, StageTesting, apperrs.ErrInvalid, false},
		{"caller may not run plays", ctxAs("u-stranger"), ticketID, StageDone, apperrs.ErrForbidden, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDecisionsFixture()
			f.targets.setTicketStage(ticketID, tt.stage)
			_, err := f.runner.RetryDecisionsCheck(tt.ctx, tt.ticket, ViaWeb)
			require.ErrorIs(t, err, tt.want)
			assert.Equal(t, tt.records, len(f.trails.all()) > 0)
		})
	}
}

func TestDecisionsCheckRun_HTTPAndMCP(t *testing.T) {
	f := newDecisionsFixture()
	h := NewRunHandler(f.runner).Routes()
	rec := do(t, h, http.MethodPost, "/api/plays/decisions-check", `{"ticket_id":"`+ticketID+`"}`, starter)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	<-f.turns.done
	assert.Equal(t, http.StatusBadRequest, do(t, h, http.MethodPost, "/api/plays/decisions-check", `{`, starter).Code)
	assert.Equal(t, http.StatusForbidden, do(t, h, http.MethodPost, "/api/plays/decisions-check", `{"ticket_id":"`+ticketID+`"}`, "u-stranger").Code)

	f2 := newDecisionsFixture()
	tools := RunMCPTools(f2.runner)
	out, err := callTool(t, tools, ctxAs(starter), "decisions_check_run", `{"ticket_id":"`+ticketID+`"}`)
	require.NoError(t, err)
	<-f2.turns.done
	assert.Equal(t, ViaMCP, out.(trailSummary).Via)
	_, err = callTool(t, tools, ctxAs(starter), "decisions_check_run", `{}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
