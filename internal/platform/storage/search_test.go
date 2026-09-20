package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/tickets"
)

func TestFtsQuery(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"single term", "sqlite", `"sqlite"*`},
		{"multiple terms", "go event bus", `"go"* AND "event"* AND "bus"*`},
		{"punctuation stripped", "build,deploy!", `"build"* AND "deploy"*`},
		{"quotes treated as separators", `say "hi"`, `"say"* AND "hi"*`},
		{"empty string", "", `""`},
		{"only punctuation", "?!?!", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ftsQuery(tt.in))
		})
	}
}

func TestSearchDocs_NoMatch_Empty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	got, err := s.Docs.Search(context.Background(), "nonexistenttermxyz", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchDocs_EmptyQuery_Empty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	got, err := s.Docs.Search(context.Background(), "", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchDocs_PunctuationOnly_NoError(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	got, err := s.Docs.Search(context.Background(), "!!!", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchDocs_RanksByRelevance(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), &docs.Doc{
		ID: "doc-sparse", Title: "one storage mention", Body: "storage appears here once",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.Docs.Create(context.Background(), &docs.Doc{
		ID: "doc-dense", Title: "storage storage storage", Body: "storage storage storage and more storage",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	got, err := s.Docs.Search(context.Background(), "storage", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "doc-dense", got[0].ID, "higher term frequency ranks first")
}

func TestSearchDocs_MatchesTitleOrBody(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), &docs.Doc{
		ID: "doc-title", Title: "migration runner", Body: "nothing about migrations here",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	got, err := s.Docs.Search(context.Background(), "runner", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "doc-title", got[0].ID)
}

func TestSearchDocs_PrefixMatchesWhileTyping(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	// The @ picker searches on every keystroke, so a single leading letter must already hit ("s" → "Storage Spine").
	got, err := s.Docs.Search(context.Background(), "s", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "doc-1", got[0].ID)
}

func TestSearchDocs_TracksUpdates(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))

	d := newTestDoc("doc-1")
	d.Body = "spine is now postgresql free"
	d.Version = 2
	require.NoError(t, s.Docs.Update(context.Background(), d))

	gone, err := s.Docs.Search(context.Background(), "sqlite", 10)
	require.NoError(t, err)
	assert.Empty(t, gone, "old body no longer indexed")

	found, err := s.Docs.Search(context.Background(), "postgresql", 10)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "doc-1", found[0].ID)
}

func TestSearchDocs_TracksDeletes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Docs.Delete(context.Background(), "doc-1"))

	got, err := s.Docs.Search(context.Background(), "sqlite", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSearchTickets_RanksByRelevance(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), &tickets.Ticket{
		ID: "t-1", Title: "ticket storage note", Body: "storage once",
		Status: tickets.StatusOpen, DocID: "doc-1", ProjectID: "project-general", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, s.Tickets.Create(context.Background(), &tickets.Ticket{
		ID: "t-2", Title: "storage storage ticket", Body: "storage storage storage",
		Status: tickets.StatusOpen, DocID: "doc-1", ProjectID: "project-general", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	got, err := s.Tickets.Search(context.Background(), "storage", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "t-2", got[0].ID)
}

func TestSearchTickets_NoMatch_Empty(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	require.NoError(t, s.Docs.Create(context.Background(), newTestDoc("doc-1")))
	require.NoError(t, s.Tickets.Create(context.Background(), &tickets.Ticket{
		ID: "t-1", Title: "ticket", Body: "body", Status: tickets.StatusOpen,
		DocID: "doc-1", ProjectID: "project-general", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	got, err := s.Tickets.Search(context.Background(), "nonexistenttermxyz", 10)
	require.NoError(t, err)
	assert.Empty(t, got)
}
