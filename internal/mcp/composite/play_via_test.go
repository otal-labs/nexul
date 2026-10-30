package composite

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/tickets"
)

// fakeRuns holds one running play run, found only for its starter on its target, as the trail lookup does.
type fakeRuns struct {
	starter, targetType, targetID string
	err                           error
}

func (f fakeRuns) RunningOn(_ context.Context, userID, targetType, targetID string) (string, string, bool, error) {
	if f.err != nil {
		return "", "", false, f.err
	}
	if userID != f.starter || targetType != f.targetType || targetID != f.targetID {
		return "", "", false, nil
	}
	return "Implement with AI", "trail-1", true, nil
}

func TestTicketUpdate_InsideAPlayRun_NamesThePlayOnTheVersionAndTheChanges(t *testing.T) {
	f := newFixture(t)
	f.tickets.SetRunningPlays(fakeRuns{starter: "u-1", targetType: "ticket", targetID: "t-1"})

	_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","title":"Login times out on 3G","body":"Rewritten.","status_id":"st-doing"}`)
	require.NoError(t, err)

	versions := f.w.versions["t-1"]
	assert.Equal(t, "Implement with AI", versions[len(versions)-1].Via)
	assert.Equal(t, []string{"u-1"}, versions[len(versions)-1].EditorIDs, "the person is still credited")
	want := tickets.Actor{Kind: tickets.ActorKindPlayMCP, PlayLabel: "Implement with AI", TrailID: "trail-1", UserID: "u-1"}
	require.NotEmpty(t, f.w.changes["t-1"])
	for _, c := range f.w.changes["t-1"] {
		assert.Equal(t, want, c.Actor, c.Field)
	}
	require.Len(t, f.w.moves, 1)
	assert.Equal(t, want, f.w.moves[0].Payload.(tickets.StatusChangedEvent).Actor)
}

func TestTicketUpdate_WithoutTheCallersOwnLiveRun_StaysUserMCP(t *testing.T) {
	tests := []struct {
		name string
		runs fakeRuns
	}{
		{"another person's run on the ticket", fakeRuns{starter: "u-2", targetType: "ticket", targetID: "t-1"}},
		{"a run on another ticket", fakeRuns{starter: "u-1", targetType: "ticket", targetID: "t-2"}},
		{"a failed lookup", fakeRuns{err: errors.New("storage is down")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.tickets.SetRunningPlays(tt.runs)

			_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","title":"Renamed","body":"Rewritten."}`)
			require.NoError(t, err, "the lookup never fails the save")

			versions := f.w.versions["t-1"]
			assert.Equal(t, "mcp", versions[len(versions)-1].Via)
			require.NotEmpty(t, f.w.changes["t-1"])
			assert.Equal(t, tickets.Actor{Kind: tickets.ActorKindUserMCP, UserID: "u-1"}, f.w.changes["t-1"][0].Actor)
		})
	}
}
