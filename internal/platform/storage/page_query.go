package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/paging"
)

// pageQuery is a list statement assembled from the filters a call sets. It reads one window of ids and the count of
// every match, and the caller fetches those rows by id through sqlc, so the column mapping stays generated.
// hand-written: sqlc cannot express a WHERE clause whose conditions are optional.
type pageQuery struct {
	from string
	// fromArgs bind the placeholders in from, which come before every condition's.
	fromArgs []any
	id       string
	order    string
	conds    []string
	args     []any
}

// where adds a condition the rows must meet, with the values its placeholders bind in order.
func (q *pageQuery) where(cond string, args ...any) {
	q.conds = append(q.conds, cond)
	q.args = append(q.args, args...)
}

func (q *pageQuery) clause() string {
	if len(q.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(q.conds, " AND ")
}

// page reads the ids of window w in order, and how many rows match in all.
func (q *pageQuery) page(ctx context.Context, db *sql.DB, w paging.Window) (ids []string, total int, err error) {
	args := append(slices.Clip(q.fromArgs), q.args...)
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+q.from+q.clause(), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total <= w.Offset {
		return []string{}, total, nil
	}
	stmt := "SELECT " + q.id + " FROM " + q.from + q.clause() + " ORDER BY " + q.order + " LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, stmt, append(args, w.Limit, w.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()
	ids = make([]string, 0, w.Limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

// ids reads every matching id in order, for a list whose rows a per-row check filters before it can be paged.
func (q *pageQuery) ids(ctx context.Context, db *sql.DB) (ids []string, err error) {
	stmt := "SELECT " + q.id + " FROM " + q.from + q.clause() + " ORDER BY " + q.order
	rows, err := db.QueryContext(ctx, stmt, append(slices.Clip(q.fromArgs), q.args...)...)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()
	ids = []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// inOrder returns items in the order of ids, the order a page query read them in; a fetch by id comes back unordered.
func inOrder[T any](ids []string, items []T, id func(T) string) []T {
	byID := make(map[string]T, len(items))
	for _, item := range items {
		byID[id(item)] = item
	}
	out := make([]T, 0, len(ids))
	for _, i := range ids {
		if item, ok := byID[i]; ok {
			out = append(out, item)
		}
	}
	return out
}
