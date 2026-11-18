package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/otal-labs/nexul/internal/plays"
)

// memoryRead is one memory a stored run's step read.
type memoryRead struct {
	trailID, projectID, memoryID string
	at                           int64
}

// backfillTrailMemoryReads writes the memories the stored runs read (ADR 0196), from one pass over the steps whose
// tool ends in memory_get, matched as a live step is; a memory deleted since is left out, as it has no title to keep.
func backfillTrailMemoryReads(ctx context.Context, tx *sql.Tx) error {
	reads, err := storedMemoryReads(ctx, tx)
	if err != nil {
		return err
	}
	for _, r := range reads {
		// hand-written: a migration step reads the schema as of its own migration, which the generated queries don't
		_, err := tx.ExecContext(ctx, `INSERT INTO trail_memory_reads (trail_id, memory_id, title, read_at)
			SELECT ?, m.id, m.title, ? FROM memories m WHERE m.id = ? AND m.project_id = ?
			ON CONFLICT (trail_id, memory_id) DO NOTHING`, r.trailID, r.at, r.memoryID, r.projectID)
		if err != nil {
			return fmt.Errorf("record trail %s read memory %s: %w", r.trailID, r.memoryID, err)
		}
	}
	return nil
}

// storedMemoryReads lists every stored memory_get step's read, each trail's in step order.
func storedMemoryReads(ctx context.Context, tx *sql.Tx) (out []memoryRead, err error) {
	// hand-written: a migration step reads the schema as of its own migration; a step stored as a plain line is no object
	rows, err := tx.QueryContext(ctx, `SELECT t.id, t.project_id, j.value FROM play_trails t, json_each(t.activity) j
		WHERE CASE WHEN j.type = 'object' THEN json_extract(j.value, '$.tool') END LIKE '%memory_get'
		ORDER BY t.id, j.key`)
	if err != nil {
		return nil, fmt.Errorf("find stored memory reads: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
		}
	}()
	for rows.Next() {
		var r memoryRead
		var raw string
		if err := rows.Scan(&r.trailID, &r.projectID, &raw); err != nil {
			return nil, fmt.Errorf("scan stored step: %w", err)
		}
		var e plays.ActivityEntry
		if json.Unmarshal([]byte(raw), &e) != nil {
			continue
		}
		id, ok := plays.MemoryReadOf(e)
		if !ok || r.projectID == "" {
			continue
		}
		r.memoryID, r.at = id, e.At.Unix()
		out = append(out, r)
	}
	return out, rows.Err()
}
