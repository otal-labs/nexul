package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tenancy"
	"github.com/otal-labs/nexul/internal/workspace"
)

// encodeCounted counts how often a frame's payload is encoded, which the audience does for any payload that is not
// already JSON.
type encodeCounted struct {
	TicketID string `json:"ticket_id"`
	encodes  *atomic.Int64
}

func (e encodeCounted) MarshalJSON() ([]byte, error) {
	e.encodes.Add(1)
	return json.Marshal(struct {
		TicketID string `json:"ticket_id"`
	}{e.TicketID})
}

// liveFrameCost sends a ticket frame and a conversation frame to n readers, the way the hub does: each frame prepared
// once, then checked for every reader. The first round fills the audience's memo with each reader's access, so the
// second costs only what the audience reads for the frame itself; that is what it returns, with how often the typed
// payload was encoded.
func liveFrameCost(t *testing.T, n int) (statements, encodes int64) {
	t.Helper()
	f, st, _ := newCountedFixtureDB(t)
	ctx := context.Background()
	now := time.Now()
	viewers := make([]string, n)
	for i := range viewers {
		viewers[i] = fmt.Sprintf("u-viewer-%d", i)
		_, _, err := f.store.Users.UpsertUser(ctx, &auth.Identity{UserID: viewers[i], Provider: auth.ProviderGitHub, ProviderUserID: viewers[i], Login: viewers[i]})
		require.NoError(t, err)
		require.NoError(t, f.store.WorkspaceMembers.AddMember(ctx, &tenancy.Member{UserID: viewers[i], WorkspaceID: wsDefault, RoleID: "role-reader", CreatedAt: now}))
	}

	audience := memoizedAudience(f.svc)
	var encoded atomic.Int64
	ticket := encodeCounted{TicketID: f.ticket.ID, encodes: &encoded}
	message := json.RawMessage(fmt.Sprintf(`{"message":{"conversation_id":%q}}`, f.channel.ID))
	round := func() {
		ticketCheck, messageCheck := audience(workspace.TopicTicketCategoryChanged, ticket), audience(chat.TopicMessageCreated, message)
		for _, viewer := range viewers {
			viewerCtx := identity.WithActor(ctx, identity.Actor{ID: viewer})
			assert.True(t, ticketCheck(viewerCtx), viewer)
			assert.True(t, messageCheck(viewerCtx), viewer)
		}
	}
	round()
	st.Reset()
	encoded.Store(0)
	round()
	return st.Count(), encoded.Load()
}

// TestStatements_LiveFrameCostsTheSameAtAnyAudienceSize: a frame naming a ticket and one naming a conversation load
// each row once for the frame, and encode a typed payload once, whether one person is connected or twenty.
func TestStatements_LiveFrameCostsTheSameAtAnyAudienceSize(t *testing.T) {
	t.Parallel()
	one, oneEncodes := liveFrameCost(t, 1)
	many, manyEncodes := liveFrameCost(t, 20)
	assert.Equal(t, int64(3), one, "the ticket with its labels, and the conversation")
	assert.Equal(t, one, many, "twenty readers")
	assert.Equal(t, int64(1), oneEncodes)
	assert.Equal(t, oneEncodes, manyEncodes, "twenty readers")
}

// TestLiveFrame_ReadsTheRowAnewOnEachFrame: a commit between two frames shows in the second, since a frame's loads
// live only as long as the frame.
func TestLiveFrame_ReadsTheRowAnewOnEachFrame(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	audience := memoizedAudience(f.svc)
	frame := json.RawMessage(fmt.Sprintf(`{"ticket_id":%q}`, f.ticket.ID))
	reads := func() bool { return audience(workspace.TopicTicketCategoryChanged, frame)(as(uReader)) }

	assert.True(t, reads())
	require.NoError(t, f.svc.ticketsSvc.Delete(as(uOwner), f.ticket.ID))
	assert.False(t, reads(), "the ticket is gone")
}
