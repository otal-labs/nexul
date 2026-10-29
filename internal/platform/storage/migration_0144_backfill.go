package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// memorySizeTable is memory_versions as migration 0144 saw it; its words are counted like a doc's.
var memorySizeTable = versionSizeTable{"memory_versions", "memory_id", "title", docVersionText}

// backfillMemoryHistory credits each kept memory version to its author, prunes every memory to its newest 10 versions
// and sizes what is left against its kept predecessor, so no kept row is sized against one the prune removed.
func backfillMemoryHistory(ctx context.Context, tx *sql.Tx) error {
	// hand-written: a migration step reads the schema as of its own migration, which the generated queries don't
	if _, err := tx.ExecContext(ctx, `UPDATE memory_versions SET editor_ids = json_array(author_id) WHERE author_id != ''`); err != nil {
		return fmt.Errorf("credit memory versions: %w", err)
	}
	// hand-written: see above
	_, err := tx.ExecContext(ctx, `DELETE FROM memory_versions WHERE id IN (
		SELECT id FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY memory_id ORDER BY version DESC) AS rank FROM memory_versions) WHERE rank > 10)`)
	if err != nil {
		return fmt.Errorf("prune memory versions: %w", err)
	}
	return backfillSizes(ctx, tx, []versionSizeTable{memorySizeTable})
}
