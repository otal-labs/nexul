package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tickets"
)

// dueFrom and dueTo are the window the due tests read.
const (
	dueFrom = "2026-10-01"
	dueTo   = "2026-10-31"
)

// withDueDates gives every ticket the fixture holds a due date, three of them outside dueFrom..dueTo, one left without.
func withDueDates(t *testing.T, f permFixture) {
	t.Helper()
	all, err := f.svc.ticketsSvc.List(as(uOwner))
	require.NoError(t, err)
	require.Greater(t, len(all), 2, "the fixture holds tickets in more than one project")
	for i, tk := range all {
		due := fmt.Sprintf("2026-10-%02d", 1+i%28)
		switch i % 5 {
		case 3:
			due = "2026-11-02"
		case 4:
			due = ""
		}
		_, err := f.svc.ticketsSvc.SetDueOn(as(uOwner), tk.ID, due)
		require.NoError(t, err)
	}
}

// rowByRowDue is what the row-by-row read check shows user of the tickets due in range: every ticket they may read, by
// due date then id.
func rowByRowDue(t *testing.T, f permFixture, user string) []string {
	t.Helper()
	all, err := f.svc.ticketsSvc.List(as(user))
	require.NoError(t, err)
	var due []*tickets.Ticket
	for _, tk := range all {
		if tk.DueOn != "" && tk.DueOn >= dueFrom && tk.DueOn <= dueTo {
			due = append(due, tk)
		}
	}
	slices.SortFunc(due, func(a, b *tickets.Ticket) int { return strings.Compare(a.DueOn+" "+a.ID, b.DueOn+" "+b.ID) })
	return ticketIDs(due)
}

func TestIntegration_TicketsDue_ShowEachViewerWhatTheRowByRowCheckShowed(t *testing.T) {
	t.Parallel()
	f := listParityFixture(t)
	withDueDates(t, f)

	sawAny := false
	for _, user := range []string{uOwner, uReader, uWriter, uPlain, uOverwrite, uOutsider, uClient, uUnlisted, uNoRole} {
		t.Run(user, func(t *testing.T) {
			want := rowByRowDue(t, f, user)
			ctx := access.WithMemo(identity.WithActor(t.Context(), identity.Actor{ID: user}), f.svc.accessSvc.NewMemo())
			got, err := f.svc.ticketsSvc.DueBetween(ctx, wsDefault, dueFrom, dueTo)
			if err != nil {
				require.ErrorIs(t, err, apperrs.ErrNotFound, "a stranger to the workspace is refused as if it did not exist")
				assert.Empty(t, want)
				return
			}
			assert.Equal(t, want, ticketIDs(got))
			sawAny = sawAny || len(got) > 0
		})
	}
	t.Cleanup(func() { assert.True(t, sawAny, "the fixture shows at least one viewer a due ticket") })

	t.Run("a project the viewer cannot open is left out", func(t *testing.T) {
		ownerSees := rowByRowDue(t, f, uOwner)
		clientSees := rowByRowDue(t, f, uClient)
		require.NotEmpty(t, clientSees)
		assert.Less(t, len(clientSees), len(ownerSees))
	})
}

func TestIntegration_TicketsDue_Route(t *testing.T) {
	t.Parallel()
	f := listParityFixture(t)
	withDueDates(t, f)
	h := tickets.NewHandler(f.svc.ticketsSvc).Routes()
	get := func(user, query string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/tickets/due?"+query, nil)
		req = req.WithContext(access.WithMemo(identity.WithActor(req.Context(), identity.Actor{ID: user}), f.svc.accessSvc.NewMemo()))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	query := "workspace_id=" + wsDefault + "&from=" + dueFrom + "&to=" + dueTo

	for _, user := range []string{uOwner, uClient} {
		rec := get(user, query)
		require.Equal(t, http.StatusOK, rec.Code, user)
		var body struct {
			Items []struct {
				ID    string `json:"id"`
				DueOn string `json:"due_on"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		var ids []string
		for _, it := range body.Items {
			ids = append(ids, it.ID)
			assert.GreaterOrEqual(t, it.DueOn, dueFrom)
			assert.LessOrEqual(t, it.DueOn, dueTo)
		}
		assert.Equal(t, rowByRowDue(t, f, user), ids, user)
	}
	assert.Equal(t, http.StatusNotFound, get(uOutsider, query).Code, "someone outside the workspace")
	assert.Equal(t, http.StatusBadRequest, get(uOwner, "workspace_id="+wsDefault+"&from=2026-01-01&to=2026-12-31").Code, "a year is too wide")
}
