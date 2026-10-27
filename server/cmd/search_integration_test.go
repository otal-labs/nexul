package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/deploy"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/mentions"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
)

// TestIntegration_SearchBox_FindsEveryKindTheCallerCanOpenInTheWorkspace runs the search box's fan-out over the wired
// services: one query reaches every kind, a ticket matched only in its body is marked so, another workspace's content
// never appears, and a caller without a kind's read permission finds none of it.
func TestIntegration_SearchBox_FindsEveryKindTheCallerCanOpenInTheWorkspace(t *testing.T) {
	t.Parallel()
	f := newPermFixture(t)
	owner := as(uOwner)
	now := time.Now()

	titled, err := f.svc.ticketsSvc.Create(owner, pGeneral, "Checkout flow", "", "", "")
	require.NoError(t, err)
	bodied, err := f.svc.ticketsSvc.Create(owner, pGeneral, "Payments", "the checkout page times out", "", "")
	require.NoError(t, err)
	require.NoError(t, f.store.Docs.Create(owner, &docs.Doc{ID: "doc-checkout", ProjectID: pGeneral, Title: "Checkout design", Body: `{"type":"doc","content":[]}`, Version: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, f.store.Tickets.LinkPR(owner, titled.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 9, Title: "Speed up checkout"}, tickets.PRStateOpen))
	require.NoError(t, f.store.Stacks.Create(owner, &deploy.Stack{ID: "stack-checkout", ProjectID: pGeneral, Name: "checkout", Slug: "checkout", Machine: "m1", Strategy: deploy.StrategyRun, CreatedAt: now, UpdatedAt: now}))
	channel, err := f.svc.chatSvc.CreateChannel(owner, wsDefault, uOwner, "checkout")
	require.NoError(t, err)
	memory, err := f.svc.memoriesSvc.Create(owner, pGeneral, "Checkout runbook", "When checkout breaks", "", false, "")
	require.NoError(t, err)
	automation, _, err := f.svc.automationsSvc.Create(owner, uOwner, wsDefault, "Checkout alerts", []string{"tickets:read"})
	require.NoError(t, err)

	other, err := f.svc.tenancySvc.Create(owner, uOwner, "Globex")
	require.NoError(t, err)
	otherProject, err := f.svc.workspaceSvc.Create(owner, other.ID, "Shop", "SHOP", "")
	require.NoError(t, err)
	_, err = f.svc.ticketsSvc.Create(owner, otherProject.ID, "Checkout elsewhere", "", "", "")
	require.NoError(t, err)

	type hit struct{ kind, id string }
	find := func(t *testing.T, user string) []hit {
		t.Helper()
		found, err := f.svc.mentionsSvc.Find(as(user), "checkout", wsDefault, "", 10)
		require.NoError(t, err)
		var out []hit
		for _, h := range found.Hits {
			out = append(out, hit{h.Type, h.ID})
		}
		return out
	}

	t.Run("an owner finds every kind, in the box's order", func(t *testing.T) {
		assert.Equal(t, []hit{
			{"ticket", titled.ID}, {"ticket", bodied.ID}, {"doc", "doc-checkout"}, {"pull_request", "acme/app#9"},
			{"stack", "stack-checkout"}, {"automation", automation.ID}, {"channel", channel.ID}, {"memory", memory.ID},
		}, find(t, uOwner))
	})

	t.Run("a body-only ticket is marked and a pull request names its ticket", func(t *testing.T) {
		found, err := f.svc.mentionsSvc.Find(owner, "checkout", wsDefault, "ticket", 10)
		require.NoError(t, err)
		require.Len(t, found.Hits, 2)
		assert.False(t, found.Hits[0].InBody)
		assert.True(t, found.Hits[1].InBody)
		prs, err := f.svc.mentionsSvc.Find(owner, "speed", wsDefault, "pull_request", 10)
		require.NoError(t, err)
		require.Len(t, prs.Hits, 1)
		assert.Equal(t, titled.ID, prs.Hits[0].TicketID)
		assert.Equal(t, "app#9", prs.Hits[0].Hint)
	})

	t.Run("a member without reads finds only the public channel", func(t *testing.T) {
		assert.Equal(t, []hit{{"channel", channel.ID}}, find(t, uPlain))
	})

	t.Run("an outsider finds nothing", func(t *testing.T) {
		assert.Empty(t, find(t, uOutsider))
	})

	t.Run("a workspace is required", func(t *testing.T) {
		_, err := f.svc.mentionsSvc.Find(owner, "checkout", "", "", 10)
		require.ErrorIs(t, err, apperrs.ErrInvalid)
	})

	t.Run("the gateway answers with the hits and the kinds cut short", func(t *testing.T) {
		routes := mentions.NewHandler(f.svc.mentionsSvc).Routes()
		req := httptest.NewRequest(http.MethodGet, "/api/search?q=checkout&limit=1&workspace_id="+wsDefault, nil).WithContext(owner)
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got mentions.Found
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Len(t, got.Hits, 8, "one title hit per kind plus the ticket found in its body")
		assert.Empty(t, got.More)
	})
}
