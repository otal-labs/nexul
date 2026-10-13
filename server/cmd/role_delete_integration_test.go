package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tenancy"
)

// TestIntegration_DeletingAHeldRole_UpdatesEachHolder: a role frame names no holder, so deleting a role people hold
// also sends a member update per holder in the same commit, which reaches whoever has Team open.
func TestIntegration_DeletingAHeldRole_UpdatesEachHolder(t *testing.T) {
	f := newPermFixture(t)
	ctx := context.Background()

	require.NoError(t, f.svc.rolesSvc.Delete(as(uOwner), wsDefault, "role-plain", uOwner))

	rows, err := f.db.Query(`SELECT CAST(payload AS TEXT) FROM outbox WHERE topic = ?`, tenancy.TopicWorkspaceMemberUpdated)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	var updated []string
	for rows.Next() {
		var raw string
		require.NoError(t, rows.Scan(&raw))
		var e tenancy.MemberEvent
		require.NoError(t, json.Unmarshal([]byte(raw), &e))
		assert.Equal(t, wsDefault, e.WorkspaceID)
		assert.Equal(t, uOwner, e.ActorID)
		assert.Contains(t, e.ProjectIDs, pGeneral, "the role may have let them into any project")
		updated = append(updated, e.UserID)
	}
	require.NoError(t, rows.Err())
	assert.ElementsMatch(t, []string{uPlain, uOverwrite}, updated, "one per holder, nobody else")

	frame, err := json.Marshal(tenancy.MemberEvent{UserID: uPlain, WorkspaceID: wsDefault})
	require.NoError(t, err)
	audience := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}
	assert.True(t, audience.allows(identity.WithActor(ctx, identity.Actor{ID: uManager}), tenancy.TopicWorkspaceMemberUpdated, json.RawMessage(frame)))
	assert.False(t, audience.allows(identity.WithActor(ctx, identity.Actor{ID: uOutsider}), tenancy.TopicWorkspaceMemberUpdated, json.RawMessage(frame)))
}
