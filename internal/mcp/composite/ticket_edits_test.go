package composite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
)

func TestTicketUpdate_Edits_SaveTheEditedBodyAsOneVersionBesideOtherFields(t *testing.T) {
	f := newFixture(t)

	got, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","status_id":"st-doing","edits":[{"find":"slow","replace":"flaky"},{"find":"flaky networks","replace":"flaky mobile networks"}]}`)

	require.NoError(t, err)
	res := got.(ticketUpdateResult)
	assert.Equal(t, "On flaky mobile networks.", res.Ticket.Body)
	assert.Equal(t, []string{"body", "status_id"}, res.Applied)
	assert.Equal(t, tickets.Status("st-doing"), f.w.tickets["t-1"].Status)
	versions := f.w.versions["t-1"]
	require.Len(t, versions, 2, "one version for the whole call")
	assert.Equal(t, "mcp", versions[1].Via)
}

func TestTicketUpdate_Edits_WhenRefused_SaveNothing(t *testing.T) {
	for name, tt := range map[string]struct{ args, wantMsg string }{
		"a find that matches nothing, after an edit that fits": {`"edits":[{"find":"slow","replace":"flaky"},{"find":"fast","replace":"x"}]`, "edit 2: find matches nothing"},
		"a find that matches twice":                            {`"edits":[{"find":"n","replace":"m"}]`, "edit 1: find matches"},
		"an empty find":                                        {`"edits":[{"find":"","replace":"x"}]`, "edit 1: find is empty"},
		"the body beside them":                                 {`"body":"new","edits":[{"find":"slow","replace":"flaky"}]`, "not both"},
		"a revert beside them":                                 {`"revert_to_version":1,"edits":[{"find":"slow","replace":"flaky"}]`, "send it alone"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)

			_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","status_id":"st-doing",`+tt.args+`}`)

			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.ErrorContains(t, err, tt.wantMsg)
			assert.Equal(t, "On slow networks.", f.w.tickets["t-1"].Body)
			assert.Equal(t, tickets.Status("st-todo"), f.w.tickets["t-1"].Status)
			assert.Len(t, f.w.versions["t-1"], 1)
		})
	}
}
