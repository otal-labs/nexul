package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// backfillDocSnippets writes every stored doc's list snippet, as a save would.
func backfillDocSnippets(ctx context.Context, tx *sql.Tx) error {
	snippets, err := storedDocSnippets(ctx, tx)
	if err != nil {
		return err
	}
	for id, snippet := range snippets {
		// hand-written: a migration step reads the schema as of its own migration, which the generated queries don't
		if _, err := tx.ExecContext(ctx, `UPDATE docs SET snippet = ? WHERE id = ?`, snippet, id); err != nil {
			return fmt.Errorf("backfill snippet of doc %s: %w", id, err)
		}
	}
	return nil
}

// storedDocSnippets reads each doc's body once and keeps only its snippet, so the bodies are never all held at once.
func storedDocSnippets(ctx context.Context, tx *sql.Tx) (out map[string]string, err error) {
	// hand-written: a migration step reads the schema as of its own migration
	rows, err := tx.QueryContext(ctx, `SELECT id, body FROM docs`)
	if err != nil {
		return nil, fmt.Errorf("read doc bodies: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
		}
	}()
	out = map[string]string{}
	for rows.Next() {
		var id, body string
		if err := rows.Scan(&id, &body); err != nil {
			return nil, fmt.Errorf("scan doc body: %w", err)
		}
		out[id] = docSnippet(body)
	}
	return out, rows.Err()
}
