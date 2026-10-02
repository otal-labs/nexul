package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/tickets"
)

type ftsHit struct {
	id    string
	title string
	rank  float64
}

func (r *DocsRepo) Search(ctx context.Context, query string, limit int) ([]docs.SearchResult, error) {
	hits, err := queryFTS(ctx, r.db, "docs_fts", "docs", "id", query, limit, true)
	if err != nil {
		return nil, fmt.Errorf("search docs: %w", err)
	}
	out := make([]docs.SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, docs.SearchResult{ID: h.id, Title: h.title, Rank: h.rank})
	}
	return out, nil
}

// Search ranks title and body matches first, then tickets found only through a note's text.
func (r *TicketsRepo) Search(ctx context.Context, query string, limit int) ([]tickets.SearchResult, error) {
	hits, err := queryFTS(ctx, r.db, "tickets_fts", "tickets", "id", query, limit, false)
	if err != nil {
		return nil, fmt.Errorf("search tickets: %w", err)
	}
	// hand-written: sqlc cannot express an FTS5 table as the operand of MATCH or bm25
	// MATERIALIZED stops SQLite flattening bm25 into the GROUP BY, where FTS5 refuses to run it.
	noteHits, err := scanFTS(ctx, r.db, `WITH n AS MATERIALIZED (SELECT ticket_id, bm25(ticket_notes_fts) AS rank FROM ticket_notes_fts WHERE ticket_notes_fts MATCH ?)
		SELECT t.id, t.title, MIN(n.rank) AS best FROM n JOIN tickets t ON t.id = n.ticket_id GROUP BY t.id ORDER BY best LIMIT ?`,
		ftsQuery(query), limit)
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
		out = append(out, tickets.SearchResult{ID: h.id, Title: h.title, Rank: h.rank})
	}
	return out, nil
}

// excludeArchived adds `AND j.archived = 0` so archived docs never appear in search; tickets have no archived flag.
func queryFTS(ctx context.Context, db *sql.DB, ftsTable, joinTable, idCol, query string, limit int, excludeArchived bool) ([]ftsHit, error) {
	archived := ""
	if excludeArchived {
		archived = " AND j.archived = 0"
	}
	// hand-written: sqlc cannot express table and column names chosen at runtime
	stmt := fmt.Sprintf(
		`SELECT j.%s, j.title, bm25(%s) AS rank FROM %s JOIN %s j ON j.rowid = %s.rowid WHERE %s MATCH ?%s ORDER BY rank LIMIT ?`,
		idCol, ftsTable, ftsTable, joinTable, ftsTable, ftsTable, archived)
	return scanFTS(ctx, db, stmt, ftsQuery(query), limit)
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
		if err := rows.Scan(&h.id, &h.title, &h.rank); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func ftsQuery(q string) string {
	fields := strings.FieldsFunc(q, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
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
