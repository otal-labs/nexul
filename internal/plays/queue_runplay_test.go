package plays_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/plays"
)

const automationID = "auto-fixer"

// asAutomation is a request on an automation's token, acting as its creator.
func asAutomation(ctx context.Context, creatorID string) context.Context {
	return identity.WithActor(ctx, identity.Actor{
		ID: creatorID, Automation: &identity.AutomationRef{ID: automationID, Name: "Fixer", WorkspaceID: ws},
	})
}

func (r *rig) docPlay(t *testing.T, label string) {
	t.Helper()
	p := &plays.Play{
		ID: ids.New(), WorkspaceID: ws, Label: label, Type: plays.TypeDoc, Enabled: true, ExcludedProjectIDs: []string{},
		CreatedAt: r.clock, UpdatedAt: r.clock,
	}
	require.NoError(t, r.store.Plays.Create(t.Context(), p))
}

func TestQueuePlay_QueuesByLabelOnTheDeveloperThroughTheQueue(t *testing.T) {
	r := newRig(t, openStore(t))
	fix := r.play(t, "Fix it")
	r.ticket("t-1")

	it, err := r.Runner.QueuePlay(asAutomation(t.Context(), dev), plays.QueuePlayInput{Play: " fix IT ", TicketID: "t-1"})
	require.NoError(t, err)

	assert.Equal(t,
		[]any{plays.QueueQueued, "", dev, fix.ID, "", automationID, plays.RunOnDeveloper, plays.LevelNormal, plays.ViaAutomation, plays.MomentAutomation},
		[]any{it.Status, it.Reason, it.PersonID, it.PlayID, it.AutoPlayID, it.AutomationID, it.RunOn, it.Priority, it.Via, it.Moment})
	assert.True(t, r.Kicked())

	again, err := r.Runner.QueuePlay(asAutomation(t.Context(), dev), plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1"})
	require.NoError(t, err)
	assert.Equal(t, it.ID, again.ID, "a second call while one waits answers the waiting run")
	assert.Len(t, r.queued(t, "t-1"), 1)

	r.dispatch(t)
	started := r.only(t, "t-1")
	assert.Equal(t, plays.QueueStarted, started.Status)
	trail, err := r.store.PlayTrails.GetTrail(t.Context(), started.TrailID)
	require.NoError(t, err)
	assert.Equal(t, []any{dev, plays.ViaAutomation}, []any{trail.StarterID, trail.Via})
}

func TestQueuePlay_RunsOnTheTesterAtThePriorityAsked(t *testing.T) {
	r := newRig(t, openStore(t))
	r.play(t, "Test it")
	r.ticket("t-1")

	it, err := r.Runner.QueuePlay(asAutomation(t.Context(), dev), plays.QueuePlayInput{Play: "Test it", TicketID: "t-1", RunOn: plays.RunOnTester, Priority: plays.LevelHigh})
	require.NoError(t, err)
	assert.Equal(t, []any{tester, plays.RunOnTester, plays.LevelHigh}, []any{it.PersonID, it.RunOn, it.Priority})
}

func TestQueuePlay_RefusesWhatItCannotQueue(t *testing.T) {
	tests := []struct {
		name string
		ctx  func(context.Context) context.Context
		in   plays.QueuePlayInput
		want error
		text string
	}{
		{"a person, not an automation", func(ctx context.Context) context.Context { return as(ctx, dev) },
			plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1"}, apperrs.ErrForbidden, "only an automation"},
		{"a default automation, with no creator", func(ctx context.Context) context.Context { return asAutomation(ctx, "") },
			plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1"}, apperrs.ErrForbidden, "no creator"},
		{"no play of that name", nil, plays.QueuePlayInput{Play: "Fix it now", TicketID: "t-1"}, apperrs.ErrNotFound, `no play named "Fix it now" in this workspace`},
		{"a doc play", nil, plays.QueuePlayInput{Play: "To tickets", TicketID: "t-1"}, apperrs.ErrInvalid, "runPlay starts ticket plays"},
		{"no ticket", nil, plays.QueuePlayInput{Play: "Fix it", TicketID: "t-gone"}, apperrs.ErrNotFound, "get ticket t-gone"},
		{"no play named", nil, plays.QueuePlayInput{TicketID: "t-1"}, apperrs.ErrInvalid, "play and ticket_id are required"},
		{"run on whoever caused it", nil, plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1", RunOn: plays.RunOnCauser}, apperrs.ErrInvalid, "developer or tester"},
		{"an unknown priority", nil, plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1", Priority: "urgent"}, apperrs.ErrInvalid, "high, normal, or low"},
		{"a creator who may not run plays", func(ctx context.Context) context.Context { return asAutomation(ctx, "u-stranger") },
			plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1"}, apperrs.ErrForbidden, "plays:run required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			r.play(t, "Fix it")
			r.docPlay(t, "To tickets")
			r.ticket("t-1")
			ctx := asAutomation(t.Context(), dev)
			if tt.ctx != nil {
				ctx = tt.ctx(t.Context())
			}

			_, err := r.Runner.QueuePlay(ctx, tt.in)

			require.ErrorIs(t, err, tt.want)
			assert.ErrorContains(t, err, tt.text)
			assert.Empty(t, r.queued(t, "t-1"), "nothing is queued")
		})
	}
}

func TestQueuePlay_NobodyToRunItOn_DidntRunWithAFailedTrail(t *testing.T) {
	r := newRig(t, openStore(t))
	r.play(t, "Test it")
	r.ticket("t-1", func(f *plays.Facts) { f.Tester = "" })

	it, err := r.Runner.QueuePlay(asAutomation(t.Context(), dev), plays.QueuePlayInput{Play: "Test it", TicketID: "t-1", RunOn: plays.RunOnTester})
	require.NoError(t, err)

	assert.Equal(t, []any{plays.QueueDidntRun, "nobody to run it on: the ticket has no tester"}, []any{it.Status, it.Reason})
	trail, err := r.store.PlayTrails.GetTrail(t.Context(), r.only(t, "t-1").TrailID)
	require.NoError(t, err)
	assert.Equal(t, plays.TrailFailed, trail.State)
}

func TestQueuePlay_AtTheDailyCap_WaitsPausedAndCountsTowardIt(t *testing.T) {
	r := newRig(t, openStore(t))
	require.NoError(t, r.store.Plays.SetAutoPlayDailyCap(t.Context(), ws, 1))
	r.play(t, "Fix it")
	r.ticket("t-1")
	queue := func() *plays.QueueItem {
		it, err := r.Runner.QueuePlay(asAutomation(t.Context(), dev), plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1"})
		require.NoError(t, err)
		return it
	}
	queue()
	r.dispatch(t)
	r.endRuns(t, "t-1")

	it := queue()

	assert.Equal(t, []any{plays.QueueQueued, plays.ReasonPaused}, []any{it.Status, it.Reason})
	r.dispatch(t)
	assert.Equal(t, plays.QueueQueued, r.queued(t, "t-1")[0].Status, "the automation's own run counted toward the cap")
}

func TestQueuePlay_RecheckAtTheFront(t *testing.T) {
	tests := []struct {
		name   string
		change func(r *rig, t *testing.T, p *plays.Play)
		status plays.QueueStatus
		reason string
	}{
		{"still holds, outside the play's show-when stage", func(*rig, *testing.T, *plays.Play) {}, plays.QueueStarted, ""},
		{"play removed", func(r *rig, t *testing.T, p *plays.Play) {
			require.NoError(t, r.store.Plays.Delete(t.Context(), p.ID))
		}, plays.QueueSkipped, "the play was removed"},
		{"play disabled", func(r *rig, t *testing.T, p *plays.Play) {
			p.Enabled = false
			require.NoError(t, r.store.Plays.Update(t.Context(), p))
		}, plays.QueueSkipped, "is disabled"},
		{"developer changed", func(r *rig, _ *testing.T, _ *plays.Play) {
			r.ticket("t-1", func(f *plays.Facts) { f.Developer = tester })
		}, plays.QueueSkipped, "the developer changed"},
		{"developer excluded since", func(r *rig, _ *testing.T, p *plays.Play) {
			r.Exclude(dev, p.ID)
		}, plays.QueueDidntRun, "plays:run required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t, openStore(t))
			fix := r.play(t, "Fix it")
			r.ticket("t-1")
			_, err := r.Runner.QueuePlay(asAutomation(t.Context(), dev), plays.QueuePlayInput{Play: "Fix it", TicketID: "t-1"})
			require.NoError(t, err)
			tt.change(r, t, fix)

			r.dispatch(t)

			it := r.only(t, "t-1")
			assert.Equal(t, tt.status, it.Status)
			assert.Contains(t, it.Reason, tt.reason)
		})
	}
}

func TestQueuePlay_HTTP(t *testing.T) {
	r := newRig(t, openStore(t))
	r.play(t, "Fix it")
	r.ticket("t-1")
	routes := plays.NewRunHandler(r.Runner, nil).Routes()
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(asAutomation(t.Context(), dev), http.MethodPost, "/api/plays/queue", strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := call(`{"play":"Fix it","ticket_id":"t-1","run_on":"developer","priority":"low"}`)
	require.Equal(t, http.StatusAccepted, rec.Code)
	var run map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &run))
	assert.Equal(t, map[string]string{"id": r.only(t, "t-1").ID, "state": "queued", "reason": ""}, run)

	rec = call(`{"play":"Fix it later","ticket_id":"t-1"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), `no play named \"Fix it later\" in this workspace`)
	assert.Equal(t, http.StatusBadRequest, call(`{`).Code)
}
