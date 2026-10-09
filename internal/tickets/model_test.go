package tickets

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

func listFixture(n int) []*Ticket {
	finished := fixedNow.Add(time.Hour)
	out := make([]*Ticket, n)
	for i := range out {
		out[i] = &Ticket{
			ID: fmt.Sprintf("t-%d", i), ProjectID: "p-1", Title: fmt.Sprintf("Fix <the> & thing %d", i), Body: "a body", Status: StatusOpen,
			Number: i + 1, Developer: "lena", Reporter: Reporter{Kind: "user", Login: "sam"}, CreatedAt: fixedNow, UpdatedAt: fixedNow,
		}
		if i%3 == 0 {
			out[i].Labels = []string{"bug", "ui"}
			out[i].FinishedAt = &finished
		}
	}
	return out
}

func TestTicketsHandler_List_AllocationsDoNotRescanEachTicket(t *testing.T) {
	repo := newFakeRepo()
	for _, tk := range listFixture(200) {
		repo.tickets[tk.ID] = tk
	}
	h := NewHandler(newTestService(repo)).Routes()
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)

	allocs := testing.AllocsPerRun(20, func() { h.ServeHTTP(httptest.NewRecorder(), req) })

	assert.LessOrEqual(t, allocs/200, 6.0, "allocations per listed ticket")
}

func TestJSONList_EncodesEachTicketAsItsOwnMarshalJSONDoes(t *testing.T) {
	ts := listFixture(6)
	want, got := httptest.NewRecorder(), httptest.NewRecorder()

	httpx.WriteJSON(want, http.StatusOK, ts)
	httpx.WriteJSON(got, http.StatusOK, jsonList(ts))

	require.Contains(t, want.Body.String(), `"labels":[]`)
	assert.Equal(t, want.Body.String(), got.Body.String())
}
