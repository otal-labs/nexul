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

func (r *TicketsRepo) Search(ctx context.Context, query string, limit int) ([]tickets.SearchResult, error) {
	hits, err := queryFTS(ctx, r.db, "tickets_fts", "tickets", "id", query, limit, false)
	if err != nil {
		return nil, fmt.Errorf("search tickets: %w", err)
	}
	out := make([]tickets.SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, tickets.SearchResult{ID: h.id, Title: h.title, Rank: h.rank})
	}
	return out, nil
}

// excludeArchived adds `AND j.archived = 0` so archived docs never appear in search; tickets have no archived flag.
func queryFTS(ctx context.Context, db *sql.DB, ftsTable, joinTable, idCol, query string, limit int, excludeArchived bool) (out []ftsHit, err error) {
	archived := ""
	if excludeArchived {
		archived = " AND j.archived = 0"
	}
	// hand-written: sqlc cannot express table and column names chosen at runtime
	stmt := fmt.Sprintf(
		`SELECT j.%s, j.title, bm25(%s) AS rank FROM %s JOIN %s j ON j.rowid = %s.rowid WHERE %s MATCH ?%s ORDER BY rank LIMIT ?`,
		idCol, ftsTable, ftsTable, joinTable, ftsTable, ftsTable, archived)
	rows, err := db.QueryContext(ctx, stmt, ftsQuery(query), limit)
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
