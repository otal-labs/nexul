package storage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/eventbus/outbox"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

// publisher records relayed topics and payloads, standing in for the bus.
type publisher struct {
	mu   sync.Mutex
	seen []published
}

type published struct {
	id      string
	topic   string
	payload map[string]any
}

func (p *publisher) Publish(_ context.Context, topic string, payload any) error {
	return p.publish("", topic, payload)
}

func (p *publisher) PublishWithID(_ context.Context, id, topic string, payload any) error {
	return p.publish(id, topic, payload)
}

func (p *publisher) publish(id, topic string, payload any) error {
	var m map[string]any
	if raw, ok := payload.(json.RawMessage); ok {
		_ = json.Unmarshal(raw, &m)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen = append(p.seen, published{id: id, topic: topic, payload: m})
	return nil
}

func (p *publisher) topics() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for _, s := range p.seen {
		out = append(out, s.topic)
	}
	return out
}

func (p *publisher) count(topic string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, s := range p.seen {
		if s.topic == topic {
			n++
		}
	}
	return n
}

// runRelay drains the outbox to published and returns once all rows are marked.
func runRelay(t *testing.T, db *sql.DB, s *storage.Store, pub *publisher) {
	t.Helper()
	relay := outbox.NewRelay(s.Outbox, pub, outbox.RelayConfig{Interval: 5 * time.Millisecond, BatchSize: 10})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()

	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(1) FROM outbox WHERE published = 0`).Scan(&n))
		return n == 0
	}, 2*time.Second, 5*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

// Acceptance criterion: create doc -> FTS5 searchable -> create ticket from
// doc -> status transition -> events published via the outbox relay.
func TestIntegration_DocsToTicketFlowPublishesEvents(t *testing.T) {
	ctx := actorCtx()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	docsSvc := docs.NewService(s.Docs, allowAll{})
	ticketsSvc := tickets.NewService(s.Tickets, s.Statuses, nil)

	doc, err := docsSvc.Create(ctx, "project-general", "Storage Spine", "SQLite migrations and FTS5 indexing")
	require.NoError(t, err)

	results, err := docsSvc.Search(ctx, "fts5", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, doc.ID, results[0].ID, "created doc is FTS5 searchable")

	tk, err := ticketsSvc.Create(ctx, "project-general", "Write migrations", "Derived from the storage doc", doc.ID, "onik97")
	require.NoError(t, err)
	assert.Equal(t, doc.ID, tk.DocID)

	updated, err := ticketsSvc.UpdateStatus(ctx, tk.ID, tickets.StatusDone)
	require.NoError(t, err)
	assert.Equal(t, tickets.StatusDone, updated.Status)

	pub := &publisher{}
	runRelay(t, db, s, pub)

	topics := pub.topics()
	assert.Contains(t, topics, docs.TopicCreated)
	assert.Contains(t, topics, tickets.TopicCreated)
	assert.Contains(t, topics, tickets.TopicStatusChanged)
}

func TestIntegration_DocUpdateVersionsAndPublishes(t *testing.T) {
	ctx := actorCtx()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := docs.NewService(s.Docs, allowAll{})

	doc, err := svc.Create(ctx, "project-general", "v1", "first")
	require.NoError(t, err)
	updated, err := svc.Update(ctx, doc.ID, "v2", "second")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Version)

	versions, err := svc.ListVersions(ctx, doc.ID)
	require.NoError(t, err)
	require.Len(t, versions, 2)

	got, err := svc.GetVersion(ctx, doc.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "v1", got.Title)

	pub := &publisher{}
	runRelay(t, db, s, pub)
	assert.Contains(t, pub.topics(), docs.TopicUpdated)
}

func TestIntegration_TicketLinksPRAndFinishes(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := tickets.NewService(s.Tickets, s.Statuses, nil)

	tk, err := svc.Create(ctx, "project-general", "Fix login", "", "", "")
	require.NoError(t, err)

	ref := tickets.PRRef{Owner: "acme", Repo: "app", Number: 42, Title: "Fix login", SHA: "abc"}
	require.NoError(t, svc.LinkPR(ctx, tk.ID, ref))

	links, err := s.Tickets.ListPRLinks(ctx, tk.ID)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, 42, links[0].Number)
	assert.Equal(t, tickets.PRStateOpen, links[0].State)

	ev := eventbus.Event{
		ID:      "evt-1",
		Topic:   "git.pr_merged",
		Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":42,"title":"Fix login","head_sha":"abc","linked_ticket_ids":[]}}`),
	}
	require.NoError(t, svc.HandlePRMerged(ctx, ev))

	links, err = s.Tickets.ListPRLinks(ctx, tk.ID)
	require.NoError(t, err)
	assert.Equal(t, tickets.PRStateMerged, links[0].State)

	got, err := svc.Get(ctx, tk.ID)
	require.NoError(t, err)
	require.NotNil(t, got.FinishedAt)

	// Redelivery of the same merge keeps ticket.finished exactly-once.
	require.NoError(t, svc.HandlePRMerged(ctx, ev))

	pub := &publisher{}
	runRelay(t, db, s, pub)
	assert.Equal(t, 1, pub.count(tickets.TopicFinished))
}

func TestIntegration_TicketDevStatusBatchesAcrossTickets(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := tickets.NewService(s.Tickets, s.Statuses, nil)

	a, err := svc.Create(ctx, "project-general", "A", "", "", "")
	require.NoError(t, err)
	b, err := svc.Create(ctx, "project-general", "B", "", "", "")
	require.NoError(t, err)

	require.NoError(t, svc.LinkPR(ctx, a.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 1, Title: "Open PR"}))
	require.NoError(t, svc.LinkPR(ctx, a.ID, tickets.PRRef{Owner: "acme", Repo: "app", Number: 2, Title: "Merged PR"}))
	require.NoError(t, svc.HandlePRMerged(ctx, eventbus.Event{
		Topic:   "git.pr_merged",
		Payload: json.RawMessage(`{"owner":"acme","repo":"app","pr":{"number":2,"title":"Merged PR","head_sha":"x","linked_ticket_ids":[]}}`),
	}))

	got, err := svc.DevStatus(ctx, []string{a.ID, b.ID, "does-not-exist"})
	require.NoError(t, err)
	assert.Equal(t, tickets.DevStatusCounts{Open: 1, Merged: 1}, got[a.ID])
	assert.Equal(t, tickets.DevStatusCounts{}, got[b.ID])
	assert.Equal(t, tickets.DevStatusCounts{}, got["does-not-exist"])
}

// Acceptance criterion (ADR 0002): a ticket's manual position persists
// within its (status, category) pair, and moving it to a new status or
// category resets its position to the end of the new pair rather than
// carrying the old value over.
func TestIntegration_TicketPositionPersistsAndResetsOnMove(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	ticketsSvc := tickets.NewService(s.Tickets, s.Statuses, nil)

	now := time.Now()
	require.NoError(t, s.Categories.Create(ctx, &workspace.Category{
		ID: "cat-1", ProjectID: "project-general", Name: "Sprint 1", CreatedAt: now, UpdatedAt: now,
	}))

	a, err := ticketsSvc.Create(ctx, "project-general", "A", "", "", "")
	require.NoError(t, err)
	c, err := ticketsSvc.Create(ctx, "project-general", "C", "", "", "", tickets.CreateOptions{CategoryID: "cat-1"})
	require.NoError(t, err)

	updatedA, err := ticketsSvc.SetPosition(ctx, a.ID, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, updatedA.Position)
	gotA, err := ticketsSvc.Get(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, gotA.Position, "manual position persists")

	// A moves into cat-1, where C already occupies position 0 — A is
	// appended to the end, not carried over at 5.
	require.NoError(t, s.Categories.SetTicketCategory(ctx, a.ID, "cat-1"))
	movedA, err := ticketsSvc.Get(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "cat-1", movedA.CategoryID)
	assert.Equal(t, c.Position+1, movedA.Position, "appended after C in the new (open, cat-1) pair")

	// D moves status into an empty (in_progress, cat-1) pair and lands first.
	d, err := ticketsSvc.Create(ctx, "project-general", "D", "", "", "", tickets.CreateOptions{CategoryID: "cat-1"})
	require.NoError(t, err)
	movedD, err := ticketsSvc.UpdateStatus(ctx, d.ID, tickets.StatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, 0, movedD.Position, "first ticket in the (in_progress, cat-1) pair")

	// A then moves status into the same pair, joining after D rather than
	// carrying its (open, cat-1) position over.
	movedA2, err := ticketsSvc.UpdateStatus(ctx, a.ID, tickets.StatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, tickets.StatusInProgress, movedA2.Status)
	assert.Equal(t, movedD.Position+1, movedA2.Position, "appended after D in the (in_progress, cat-1) pair")
}

// Acceptance criterion (ADR 0025): docs.project_id is a
// real FK, so creating a doc against a project id that doesn't exist is
// rejected at the storage layer, not silently accepted.
func TestIntegration_CreateDocWithUnknownProjectIsConflict(t *testing.T) {
	ctx := actorCtx()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := docs.NewService(s.Docs, allowAll{})

	_, err := svc.Create(ctx, "does-not-exist", "orphan", "body")
	require.Error(t, err)
}

// Acceptance criterion (ticket 10): GET /api/docs (ListByProject) filters to
// one project's docs — a flat list, no Collections-equivalent sub-grouping.
func TestIntegration_ListDocsByProjectFiltersCorrectly(t *testing.T) {
	ctx := actorCtx()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	docsSvc := docs.NewService(s.Docs, allowAll{})

	now := time.Now()
	require.NoError(t, s.Projects.Create(ctx, &workspace.Project{
		ID: "project-other", Name: "Other", Position: 1, WorkspaceID: "workspace-default", CreatedAt: now, UpdatedAt: now,
	}))

	inGeneral, err := docsSvc.Create(ctx, "project-general", "In general", "body")
	require.NoError(t, err)
	_, err = docsSvc.Create(ctx, "project-other", "In other", "body")
	require.NoError(t, err)

	items, err := docsSvc.ListByProject(ctx, "project-general")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, inGeneral.ID, items[0].ID)
	assert.Equal(t, "project-general", items[0].ProjectID)
}

func TestIntegration_CreateTicketWithUnknownDocIsConflict(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := tickets.NewService(s.Tickets, s.Statuses, nil)

	_, err := svc.Create(ctx, "project-general", "orphan", "", "does-not-exist", "")
	require.Error(t, err)
}

func TestIntegration_TicketStatusChangePublishesFromStatusToStatus(t *testing.T) {
	ctx := context.Background()
	db := newDB(t)
	s := storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
	svc := tickets.NewService(s.Tickets, s.Statuses, nil)

	tk, err := svc.Create(ctx, "project-general", "Fix", "", "", "")
	require.NoError(t, err)
	_, err = svc.UpdateStatus(ctx, tk.ID, tickets.StatusInProgress)
	require.NoError(t, err)

	pub := &publisher{}
	runRelay(t, db, s, pub)

	var found bool
	for _, p := range pub.seen {
		if p.topic == tickets.TopicStatusChanged {
			payload := p.payload
			assert.Equal(t, "open", payload["from"])
			assert.Equal(t, "in_progress", payload["to"])
			found = true
		}
	}
	assert.True(t, found, "ticket.status_changed published with from/to")
}
