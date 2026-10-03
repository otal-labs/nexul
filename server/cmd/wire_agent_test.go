package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

// TestIntegration_AgentTicketReader_KeyAndMarkdownBody names the ticket by its key and hands over its editor body as
// markdown, the only shape the pipeline finds embedded images in.
func TestIntegration_AgentTicketReader_KeyAndMarkdownBody(t *testing.T) {
	ctx := context.Background()
	db := mentionsTestDB(t)
	_, err := db.Exec(`UPDATE projects SET prefix = 'GEN' WHERE id = 'project-general'`)
	require.NoError(t, err)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	ticketsSvc := tickets.NewService(s.Tickets, s.Statuses, nil)
	ticketsSvc.SetGate(allowTickets{})
	body := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"See"}]},` +
		`{"type":"image","attrs":{"src":"/api/attachments/att-1","alt":"shot"}}]}`
	created, err := ticketsSvc.Create(ctx, "project-general", "Login fails", body, "", "")
	require.NoError(t, err)

	got, err := agentTicketReader{svc: ticketsSvc, projects: s.Projects, statuses: s.Statuses}.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "GEN-1", got.Key)
	assert.Equal(t, "Login fails", got.Title)
	assert.Contains(t, got.Body, "](/api/attachments/att-1)")
}

// TestIntegration_AgentTicketReader_DoneReadsTheColumnStage reads done from the column's stage, never its name.
func TestIntegration_AgentTicketReader_DoneReadsTheColumnStage(t *testing.T) {
	ctx := t.Context()
	db := mentionsTestDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	ticketsSvc := tickets.NewService(s.Tickets, s.Statuses, nil)
	ticketsSvc.SetGate(allowTickets{})
	created, err := ticketsSvc.Create(ctx, "project-general", "Pick a strategy", "", "", "")
	require.NoError(t, err)
	reader := agentTicketReader{svc: ticketsSvc, projects: s.Projects, statuses: s.Statuses}

	got, err := reader.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, got.Done)

	_, err = db.Exec(`UPDATE tickets SET status = 'closed' WHERE id = ?`, created.ID)
	require.NoError(t, err)
	got, err = reader.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, got.Done, "Closed is a done-stage column")
}
