package composite

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/tickets"
)

func TestTicketGet_ListsTheKeptVersionsAndTheLatestTwentyChanges(t *testing.T) {
	f := newFixture(t)
	for i := range 25 {
		f.w.changes["t-1"] = append(f.w.changes["t-1"], tickets.Change{ID: fmt.Sprint(i), Field: "due_on", To: fmt.Sprintf("2026-10-%02d", i+1)})
	}
	_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","body":"Rewritten."}`)
	require.NoError(t, err)

	got, err := call(t, t.Context(), f.ticketTools(), "ticket_get", `{"id":"t-1"}`)
	require.NoError(t, err)
	d := got.(ticketDetail)
	assert.Equal(t, "Rewritten.", d.Body)
	require.Len(t, d.Versions, 2)
	assert.Equal(t, 2, d.Versions[0].Version, "newest first")
	assert.Equal(t, "mcp", d.Versions[0].Via, "an agent's save says so")
	assert.Len(t, d.Changes, 20, "the list is capped, newest first")
	assert.Zero(t, d.CurrentVersion, "no older body was asked for")
}

func TestTicketGet_WithAVersion_ReadsThatBodyAndNamesTheCurrentVersion(t *testing.T) {
	f := newFixture(t)
	_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", `{"id":"t-1","body":"Rewritten."}`)
	require.NoError(t, err)

	got, err := call(t, t.Context(), f.ticketTools(), "ticket_get", `{"id":"t-1","version":1}`)
	require.NoError(t, err)
	d := got.(ticketDetail)
	assert.Equal(t, "On slow networks.", d.Body)
	assert.Equal(t, 2, d.CurrentVersion)

	_, err = call(t, t.Context(), f.ticketTools(), "ticket_get", `{"id":"t-1","version":9}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketUpdate_RevertToVersion_RestoresThatBodyAsANewVersion(t *testing.T) {
	f := newFixture(t)
	ctx := asUser(t.Context())
	_, err := call(t, ctx, f.ticketTools(), "ticket_update", `{"id":"t-1","body":"Rewritten."}`)
	require.NoError(t, err)

	got, err := call(t, ctx, f.ticketTools(), "ticket_update", `{"id":"t-1","revert_to_version":1}`)
	require.NoError(t, err)
	res := got.(ticketUpdateResult)
	assert.Equal(t, []string{"revert_to_version"}, res.Applied)
	assert.Equal(t, "On slow networks.", res.Ticket.Body)
	versions := f.w.versions["t-1"]
	require.Len(t, versions, 3, "the restore is a new version on top")
	assert.Equal(t, "On slow networks.", versions[2].Body)
	assert.Equal(t, "mcp", versions[2].Via)

	_, err = call(t, ctx, f.ticketTools(), "ticket_update", `{"id":"t-1","revert_to_version":9}`)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestTicketUpdate_RevertToVersionWithAnotherField_IsRefusedAndChangesNothing(t *testing.T) {
	for name, args := range map[string]string{
		"a title":      `{"id":"t-1","revert_to_version":1,"title":"Renamed"}`,
		"a label":      `{"id":"t-1","revert_to_version":1,"add_labels":["ui"]}`,
		"a due date":   `{"id":"t-1","revert_to_version":1,"due_on":"2026-10-30"}`,
		"a found-in":   `{"id":"t-1","revert_to_version":1,"found_in_unknown":true}`,
		"a status too": `{"id":"t-1","revert_to_version":1,"status_id":"st-doing"}`,
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			_, err := call(t, asUser(t.Context()), f.ticketTools(), "ticket_update", args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
			assert.Contains(t, err.Error(), "send it alone")
			assert.Equal(t, "Login times out", f.w.tickets["t-1"].Title)
			assert.Equal(t, tickets.Status("st-todo"), f.w.tickets["t-1"].Status)
			assert.Len(t, f.w.versions["t-1"], 1)
		})
	}
}
