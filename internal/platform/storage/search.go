package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/tickets"
)

type ftsHit struct {
	id      string
	title   string
	rank    float64
	updated int64
	snippet string
	// body is a doc's markdown, read so the heading above a match can be found; tickets leave it empty.
	body   string
	noteID string
}

// snippetTokens is how many words of the matched text a hit carries, enough for one line of context.
const snippetTokens = 12

func (r *DocsRepo) Search(ctx context.Context, query string, limit int) ([]docs.SearchResult, error) {
	return r.SearchIn(ctx, query, "", limit)
}

// SearchIn is Search held to one workspace's projects, or every workspace when workspaceID is empty, so another
// workspace's hits never take a place in the limit.
func (r *DocsRepo) SearchIn(ctx context.Context, query, workspaceID string, limit int) ([]docs.SearchResult, error) {
	// ponytail: reads each hit's markdown to find its heading; store heading offsets if docs grow past a few hundred KB.
	hits, err := queryFTS(ctx, r.db, "docs_fts", "docs", "j.body_md", query, workspaceID, limit, true)
	if err != nil {
		return nil, fmt.Errorf("search docs: %w", err)
	}
	words := queryWords(query)
	out := make([]docs.SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, docs.SearchResult{
			ID: h.id, Title: h.title, Rank: h.rank, UpdatedAt: time.Unix(h.updated, 0).UTC(),
			Snippet: h.snippet, Heading: richtext.HeadingAbove(h.body, words),
		})
	}
	return out, nil
}

// Search ranks title and body matches first, then tickets found only through a note's text.
func (r *TicketsRepo) Search(ctx context.Context, query string, limit int) ([]tickets.SearchResult, error) {
	return r.SearchIn(ctx, query, "", limit)
}

// SearchIn is Search held to one workspace's projects, or every workspace when workspaceID is empty.
func (r *TicketsRepo) SearchIn(ctx context.Context, query, workspaceID string, limit int) ([]tickets.SearchResult, error) {
	hits, err := queryFTS(ctx, r.db, "tickets_fts", "tickets", "''", query, workspaceID, limit, false)
	if err != nil {
		return nil, fmt.Errorf("search tickets: %w", err)
	}
	// hand-written: sqlc cannot express an FTS5 table as the operand of MATCH or bm25
	// MATERIALIZED stops SQLite flattening bm25 into the GROUP BY, where FTS5 refuses to run it; MIN keeps its row's snippet.
	noteHits, err := scanFTS(ctx, r.db, `WITH n AS MATERIALIZED (SELECT ticket_id, attachment_id, bm25(ticket_notes_fts) AS rank,
			snippet(ticket_notes_fts, 2, '', '', '…', `+strconv.Itoa(snippetTokens)+`) AS snip FROM ticket_notes_fts WHERE ticket_notes_fts MATCH ?)
		SELECT t.id, t.title, MIN(n.rank) AS best, t.updated_at, n.snip, '', COALESCE((SELECT m.id FROM messages m WHERE m.attachment_id = n.attachment_id), '')
		FROM n JOIN tickets t ON t.id = n.ticket_id WHERE `+inLiveWorkspace("t")+` AND `+inWorkspace("t")+` GROUP BY t.id ORDER BY best LIMIT ?`,
		ftsQuery(query), workspaceID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("search ticket notes: %w", err)
	}
	seen := make(map[string]bool, len(hits))
	for _, h := range hits {
		seen[h.id] = true
	}
	for _, h := range noteHits {
		if len(hits) >= limit {
			break
		}
		if seen[h.id] {
			continue
		}
		hits = append(hits, h)
	}
	out := make([]tickets.SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, tickets.SearchResult{
			ID: h.id, Title: h.title, Rank: h.rank, UpdatedAt: time.Unix(h.updated, 0).UTC(), Snippet: h.snippet, NoteID: h.noteID,
		})
	}
	return out, nil
}

// ticketHits ranks the tickets a query matches, title and body matches first by relevance and then those found only
// through a note's text, as a table of (id, tier, rank) a page query reads from; its one placeholder pair is the query.
// hand-written: sqlc cannot express an FTS5 table as the operand of MATCH or bm25
const ticketHits = `(WITH body AS MATERIALIZED (SELECT j.id, bm25(tickets_fts) AS rank FROM tickets_fts JOIN tickets j ON j.rowid = tickets_fts.rowid WHERE tickets_fts MATCH ?),
	note AS MATERIALIZED (SELECT ticket_id, bm25(ticket_notes_fts) AS rank FROM ticket_notes_fts WHERE ticket_notes_fts MATCH ?)
	SELECT id, 0 AS tier, rank FROM body
	UNION ALL SELECT ticket_id, 1, MIN(rank) FROM note WHERE ticket_id NOT IN (SELECT id FROM body) GROUP BY ticket_id) h
	JOIN tickets t ON t.id = h.id`

// ticketMatches is every ticket ticketHits ranks, unranked, for counting them; its one placeholder pair is the query.
const ticketMatches = `(SELECT * FROM tickets WHERE rowid IN (SELECT rowid FROM tickets_fts WHERE tickets_fts MATCH ?)
	OR id IN (SELECT ticket_id FROM ticket_notes_fts WHERE ticket_notes_fts MATCH ?)) t`

// inLiveWorkspace holds the doc or ticket aliased alias to a project outside any archived workspace, so no hit from one takes a place in the limit.
func inLiveWorkspace(alias string) string {
	return `NOT EXISTS (SELECT 1 FROM projects p JOIN workspaces w ON w.id = p.workspace_id WHERE p.id = ` + alias + `.project_id AND w.archived_at IS NOT NULL)`
}

// inWorkspace holds the doc or ticket aliased alias to one workspace's projects; its placeholder pair is the workspace
// id, and an empty one matches every workspace.
func inWorkspace(alias string) string {
	return `(? = '' OR EXISTS (SELECT 1 FROM projects p WHERE p.id = ` + alias + `.project_id AND p.workspace_id = ?))`
}

// excludeArchived adds `AND j.archived = 0` so archived docs never appear in search; tickets have no archived flag.
// bodyCol is what a hit's body is read from, an empty string literal for none; the snippet is the second indexed column's.
func queryFTS(ctx context.Context, db *sql.DB, ftsTable, joinTable, bodyCol, query, workspaceID string, limit int, excludeArchived bool) ([]ftsHit, error) {
	archived := " AND " + inLiveWorkspace("j") + " AND " + inWorkspace("j")
	if excludeArchived {
		archived += " AND j.archived = 0"
	}
	// hand-written: sqlc cannot express table and column names chosen at runtime
	stmt := fmt.Sprintf(
		`SELECT j.id, j.title, bm25(%s) AS rank, j.updated_at, snippet(%s, 1, '', '', '…', %d), %s, ''
		FROM %s JOIN %s j ON j.rowid = %s.rowid WHERE %s MATCH ?%s ORDER BY rank LIMIT ?`,
		ftsTable, ftsTable, snippetTokens, bodyCol, ftsTable, joinTable, ftsTable, ftsTable, archived)
	return scanFTS(ctx, db, stmt, ftsQuery(query), workspaceID, workspaceID, limit)
}

func scanFTS(ctx context.Context, db *sql.DB, stmt string, args ...any) (out []ftsHit, err error) {
	rows, err := db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	for rows.Next() {
		var h ftsHit
		if err := rows.Scan(&h.id, &h.title, &h.rank, &h.updated, &h.snippet, &h.body, &h.noteID); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// queryWords splits a query as FTS5's tokenizer does, on anything that is not a letter or a digit.
func queryWords(q string) []string {
	return strings.FieldsFunc(q, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func ftsQuery(q string) string {
	fields := queryWords(q)
	if len(fields) == 0 {
		return `""`
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		// Prefix match ("t"* finds "Ticket") — the @ picker and search box both feed partial words while typing.
		parts = append(parts, `"`+strings.ReplaceAll(f, `"`, `""`)+`"*`)
	}
	return strings.Join(parts, " AND ")
}
