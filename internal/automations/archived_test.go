package automations_test

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

const archivedWorkspace = "workspace-default"

func setArchived(t *testing.T, store *storage.Store, archived bool) {
	t.Helper()
	var at *time.Time
	if archived {
		now := time.Now()
		at = &now
	}
	require.NoError(t, store.Workspaces.SetArchived(t.Context(), archivedWorkspace, at, time.Now()))
}

func workspaceEventOf(topic string) eventbus.Event {
	return eventbus.Event{ID: topic, Topic: topic, Payload: []byte(`{"workspace_id":"` + archivedWorkspace + `"}`)}
}

// A restore resumes delivery from then on; events from while it was archived are never replayed.
func TestArchivedWorkspace_AutomationGetsNothingUntilRestored(t *testing.T) {
	setup := newDialinSetup(t)
	store := storage.New(setup.db, []byte("0123456789abcdef0123456789abcdef"))
	created, token, err := setup.svc.Create(t.Context(), "owner", archivedWorkspace, "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.SetEnabled(t.Context(), "owner", created.ID, true)
	require.NoError(t, err)
	announce := automations.Frame{Subscriptions: []string{"ticket.created"}}

	conn := dialAnnounce(t, setup.wsURL(token), announce)
	now := time.Now()
	setup.seedEvent(t, "ev-live", "ticket.created", `{}`, now)
	var ev automations.Frame
	require.NoError(t, wsjson.Read(t.Context(), conn, &ev))
	require.Equal(t, "ev-live", ev.EventID, "a live workspace delivers")
	require.NoError(t, wsjson.Write(t.Context(), conn, automations.Frame{Type: automations.FrameRunFinished, RunID: ev.RunID, Outcome: automations.OutcomeSuccess}))

	setArchived(t, store, true)
	require.NoError(t, setup.svc.HandleWorkspaceArchived(t.Context(), workspaceEventOf("workspace.archived")))
	require.Error(t, wsjson.Read(t.Context(), conn, &ev), "archiving drops the connection that was handed secrets")
	_, err = setup.svc.AuthenticateToken(t.Context(), token)
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized, "its token is refused to the API")

	setup.seedEvent(t, "ev-archived", "ticket.created", `{}`, now.Add(time.Second))
	conn = dialAnnounce(t, setup.wsURL(token), announce)
	setup.seedEvent(t, "ev-archived-connected", "ticket.created", `{}`, now.Add(2*time.Second))
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	assert.Error(t, wsjson.Read(ctx, conn, &ev), "a worker that dials in anyway is delivered nothing")

	setArchived(t, store, false)
	require.NoError(t, setup.svc.HandleWorkspaceRestored(t.Context(), workspaceEventOf("workspace.restored")))
	require.Error(t, wsjson.Read(t.Context(), conn, &ev), "restoring drops it again, so it reconnects with its secrets")
	conn = dialAnnounce(t, setup.wsURL(token), announce)
	setup.seedEvent(t, "ev-restored", "ticket.created", `{}`, now.Add(3*time.Second))
	require.NoError(t, wsjson.Read(t.Context(), conn, &ev))
	assert.Equal(t, "ev-restored", ev.EventID, "delivery resumes, and nothing from the archived time is replayed")
}

// The host drops the worker on its next poll, but the worker token stops working at once.
func TestArchivedWorkspace_HostIsAssignedNothingUntilRestored(t *testing.T) {
	f := newHostsFixture(t)
	_, credential := f.enroll(t, automations.InstanceHostName)
	a, _, err := f.svc.Create(adminCtx(), "admin", archivedWorkspace, "Notify", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = f.svc.SetEnabled(adminCtx(), "admin", a.ID, true)
	require.NoError(t, err)
	token := f.tokenFor(t, credential, a.ID)
	require.Len(t, f.assignments(t, credential), 1)

	setArchived(t, f.store, true)
	assert.Empty(t, f.assignments(t, credential))
	_, err = f.svc.AuthenticateToken(t.Context(), token)
	assert.ErrorIs(t, err, apperrs.ErrUnauthorized)

	setArchived(t, f.store, false)
	assert.Len(t, f.assignments(t, credential), 1)
	_, err = f.svc.AuthenticateToken(t.Context(), token)
	assert.NoError(t, err)
}
