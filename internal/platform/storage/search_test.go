package storage

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/platform/paging"
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

func TestSearchDocs_MatchesWhatTheDocHolds(t *testing.T) {
	t.Parallel()
	richText := `{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Storage"}]},{"type":"paragraph","content":[{"type":"text","text":"SQLite is the spine"}]}]}`
	tests := []struct {
		name, title, body, query string
		found                    bool
	}{
		{"a word of the title", "migration runner", "nothing about migrations here", "runner", true},
		{"a word of a plain markdown body", "Storage Spine", "legacy markdown body about migrations", "migrations", true},
		{"a word the rich text renders", "Storage Spine", richText, "spine", true},
		{"not the keys of the rich text's JSON", "Storage Spine", richText, "content", false},
		{"the first letter of a word, as the picker types it", "Storage Spine", "SQLite, migrations, FTS5", "s", true},
		{"a term no doc holds", "Storage Spine", "SQLite, migrations, FTS5", "nonexistenttermxyz", false},
		{"an empty query", "Storage Spine", "SQLite, migrations, FTS5", "", false},
		{"punctuation only", "Storage Spine", "SQLite, migrations, FTS5", "!!!", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newTestStore(t)
			d := newTestDoc("doc-1")
			d.Title, d.Body = tt.title, tt.body
			require.NoError(t, s.Docs.Create(t.Context(), d))

			got, err := s.Docs.Search(t.Context(), tt.query, 10)
			require.NoError(t, err)
			if !tt.found {
				assert.Empty(t, got)
				return
			}
			require.Len(t, got, 1)
			assert.Equal(t, "doc-1", got[0].ID)
		})
	}
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

// seedNoteThread files ticket ticketID with a thread conv-<ticketID> and returns a note-posting helper on it.
func seedNoteThread(t *testing.T, s *Store, ticketID, title, body string) func(noteID, markdown string) *chat.Message {
	t.Helper()
	ctx := context.Background()
	tk := newTestTicket(ticketID, "")
	tk.Title, tk.Body = title, body
	require.NoError(t, s.Tickets.Create(ctx, tk))
	convID := "conv-" + ticketID
	require.NoError(t, s.Chat.CreateConversation(ctx, newTestConversation(convID, chat.KindTicketThread, "", ticketID, "u-1"), []string{"u-1"}))
	return func(noteID, markdown string) *chat.Message {
		m := &chat.Message{ID: noteID, ConversationID: convID, AuthorID: "u-1", AuthorKind: chat.AuthorAgent,
			AttachmentID: noteID + "-file", CreatedAt: chatFixedNow, UpdatedAt: chatFixedNow}
		require.NoError(t, s.Chat.CreateNote(ctx, m, &chat.NoteFile{ID: m.AttachmentID, Name: "context.md", Markdown: markdown}))
		return m
	}
}

func searchIDs(t *testing.T, s *Store, query string) []string {
	t.Helper()
	hits, err := s.Tickets.Search(context.Background(), query, 10)
	require.NoError(t, err)
	ids := make([]string, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
	}
	return ids
}

func TestSearchTickets_NoteText_FindsItsTicketAfterTitleAndBodyMatches(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	postOnBodyMatch := seedNoteThread(t, s, "t-body", "Flaky deploys", "the runner drops its websocket")
	postOnNoteOnly := seedNoteThread(t, s, "t-note", "Slow board", "columns lag")
	postOnBodyMatch("n-1", "websocket again")
	postOnNoteOnly("n-2", "# Context\n\nwebsocket websocket websocket reconnect storm")
	postOnNoteOnly("n-3", "the websocket keeps closing")

	assert.Equal(t, []string{"t-body", "t-note"}, searchIDs(t, s, "websocket"),
		"a title or body match ranks above a note-only match, and each ticket appears once")
}

func TestSearchTickets_DeletedNote_StopsMatching(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	note := seedNoteThread(t, s, "t-1", "Slow board", "columns lag")("n-1", "reconnect storm")
	require.Equal(t, []string{"t-1"}, searchIDs(t, s, "reconnect"))

	require.NoError(t, s.Chat.DeleteNote(context.Background(), note, nil, chatFixedNow))

	assert.Empty(t, searchIDs(t, s, "reconnect"))
}

func TestSearchTickets_ReplacedNoteFile_MatchesItsNewTextOnly(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	seedChatUser(t, s, "u-1")
	note := seedNoteThread(t, s, "t-1", "Slow board", "columns lag")("n-1", "reconnect storm")

	_, err := s.db.Exec(`UPDATE attachments SET data = ?, size = ? WHERE id = ?`, []byte("virtualized rows"), 16, note.AttachmentID)
	require.NoError(t, err)

	assert.Empty(t, searchIDs(t, s, "reconnect"), "the replaced text no longer matches")
	assert.Equal(t, []string{"t-1"}, searchIDs(t, s, "virtualized"))
}

func pageTicketIDs(t *testing.T, s *Store, f tickets.TicketFilter, scope tickets.TicketScope) []string {
	t.Helper()
	return pageAll(t, 30, func(offset, limit int) ([]string, int) {
		ts, total, err := s.Tickets.Page(t.Context(), f, scope, paging.Window{Offset: offset, Limit: limit})
		require.NoError(t, err)
		ids := make([]string, len(ts))
		for i, tk := range ts {
			ids[i] = tk.ID
		}
		return ids, total
	})
}

// seedPagedTickets files 230 tickets, five to a second, a third in a second project and a seventh from a doc, with
// "login" in every title, then a ticket matched by a note alone, and returns them all oldest first.
func seedPagedTickets(t *testing.T, s *Store) []*tickets.Ticket {
	t.Helper()
	ctx := t.Context()
	seedChatUser(t, s, "u-1")
	seedProject(t, s, "p-hidden", "workspace-default", "HID")
	require.NoError(t, s.Docs.Create(ctx, newTestDoc("doc-1")))
	var all []*tickets.Ticket
	for i := range 230 {
		tk := newTestTicket(fmt.Sprintf("t-%03d", i), "")
		tk.Title = strings.Repeat("login ", 1+i%4) + "times out"
		tk.CreatedAt = tk.CreatedAt.Add(time.Duration(i/5) * time.Second)
		tk.ProjectID = map[bool]string{true: "p-hidden", false: "project-general"}[i%3 == 0]
		tk.DocID = map[bool]string{true: "doc-1", false: ""}[i%7 == 0]
		require.NoError(t, s.Tickets.Create(ctx, tk))
		all = append(all, tk)
	}
	seedNoteThread(t, s, "t-note", "Slow board", "columns lag")("n-1", "login fails behind the proxy")
	noteTicket, err := s.Tickets.GetByID(ctx, "t-note")
	require.NoError(t, err)
	all = append(all, noteTicket)
	slices.SortStableFunc(all, func(a, b *tickets.Ticket) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return all
}

// TestTicketsRepo_Page_RanksAsSearchAndFiltersInSQL: past the 200 matches search used to stop at, the pages hold every
// match in Search's own order, note-only matches last, and each filter and scope keeps what the per-row rule keeps.
func TestTicketsRepo_Page_RanksAsSearchAndFiltersInSQL(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	byAge := seedPagedTickets(t, s)
	byID := map[string]*tickets.Ticket{}
	var ageOrder []string
	for _, tk := range byAge {
		byID[tk.ID] = tk
		ageOrder = append(ageOrder, tk.ID)
	}
	hits, err := s.Tickets.Search(t.Context(), "login", 1000)
	require.NoError(t, err)
	var ranked []string
	for _, h := range hits {
		ranked = append(ranked, h.ID)
	}
	require.Len(t, ranked, 231)

	for _, f := range []tickets.TicketFilter{{}, {ProjectID: "project-general"}, {DocID: "doc-1"}, {Query: "login"}, {Query: "login", DocID: "doc-1"}} {
		for _, scope := range []tickets.TicketScope{{All: true}, {ProjectIDs: []string{"project-general"}}, {ProjectIDs: []string{}}} {
			order := map[bool][]string{true: ranked, false: ageOrder}[f.Query != ""]
			var want []string
			for _, id := range order {
				tk := byID[id]
				if (scope.All || slices.Contains(scope.ProjectIDs, tk.ProjectID)) && (f.ProjectID == "" || tk.ProjectID == f.ProjectID) && (f.DocID == "" || tk.DocID == f.DocID) {
					want = append(want, id)
				}
			}
			assert.Equal(t, want, pageTicketIDs(t, s, f, scope), "%+v %+v", f, scope)
		}
	}
}

// TestTicketsRepo_Page_BlockedOnlyCountsTheBlockersTheReaderSees: a ticket is blocked while a blocker in a project the
// scope reads is neither finished nor in a done-stage column; a cleared blocker or one the reader cannot see holds nothing.
func TestTicketsRepo_Page_BlockedOnlyCountsTheBlockersTheReaderSees(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s := newTestStore(t)
	seedProject(t, s, "p-hidden", "workspace-default", "HID")
	for _, tk := range []struct{ id, project string }{{"waits-on-open", "project-general"}, {"waits-on-hidden", "project-general"}, {"waits-on-done", "project-general"}, {"waits-on-merged", "project-general"}, {"open", "project-general"}, {"hidden", "p-hidden"}, {"done", "project-general"}, {"merged", "project-general"}} {
		ticket := newTestTicket(tk.id, "")
		ticket.ProjectID = tk.project
		if tk.id == "done" {
			ticket.Status = tickets.StatusDone
		}
		require.NoError(t, s.Tickets.Create(ctx, ticket))
	}
	_, err := s.Tickets.SetFinishedAt(ctx, "merged", time.Now(), nil)
	require.NoError(t, err)
	for blocked, blocker := range map[string]string{"waits-on-open": "open", "waits-on-hidden": "hidden", "waits-on-done": "done", "waits-on-merged": "merged"} {
		require.NoError(t, s.Tickets.PutLink(ctx, tickets.TicketLink{TicketID: blocked, Kind: tickets.LinkBlockedBy, TargetID: blocker, CreatedAt: time.Now()}))
	}
	blocked := tickets.TicketFilter{BlockedOnly: true}

	assert.ElementsMatch(t, []string{"waits-on-open", "waits-on-hidden"}, pageTicketIDs(t, s, blocked, tickets.TicketScope{All: true}))
	assert.ElementsMatch(t, []string{"waits-on-open"}, pageTicketIDs(t, s, blocked, tickets.TicketScope{ProjectIDs: []string{"project-general"}}))
}
