package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// The placeholder memory every project was seeded with before the default memories, as it stood unedited.
const (
	placeholderMemoryTitle = "Working in this project"
	placeholderMemoryBody  = `This memory is Agent's shared context for this project — durable notes worth remembering across every chat turn.

Replace this starter text with anything the team wants Agent to always know: conventions, gotchas, or standing preferences.

Edit it any time; it stays always-included until someone turns that off.`
)

// placeholderOnly is an ordinary memory that is the untouched placeholder; the interview memory and decisions log have a kind.
const placeholderOnly = `kind = '' AND title = ? AND version = 1 AND body = ?`

// seedDefaultMemories gives each project with no memories of its own the shipped default memories (ADR 0194), replacing
// the untouched placeholder; the instance cannot have edited a default memory before this migration, so these are the code's.
func seedDefaultMemories(ctx context.Context, tx *sql.Tx) error {
	seeds := make([]memories.Seed, 0, len(memories.DefaultKeys))
	for _, d := range memories.Defaults() {
		seed, err := memories.SeedOf(d.Text)
		if err != nil {
			return fmt.Errorf("default memory %s: %w", d.Key, err)
		}
		seeds = append(seeds, seed)
	}
	projects, err := projectsWithoutOwnMemories(ctx, tx)
	if err != nil {
		return err
	}
	at := time.Now().Unix()
	for _, p := range projects {
		if err := replacePlaceholder(ctx, tx, p, seeds, at); err != nil {
			return fmt.Errorf("default memories for project %s: %w", p.id, err)
		}
	}
	return nil
}

type projectRef struct{ id, workspaceID string }

func projectsWithoutOwnMemories(ctx context.Context, tx *sql.Tx) (out []projectRef, err error) {
	// hand-written: a migration step reads the schema as of its own migration, which the generated queries don't
	rows, err := tx.QueryContext(ctx, `SELECT p.id, p.workspace_id FROM projects p WHERE NOT EXISTS (
		SELECT 1 FROM memories m WHERE m.project_id = p.id AND m.kind = '' AND NOT (`+placeholderOnly+`)) ORDER BY p.id`,
		placeholderMemoryTitle, placeholderMemoryBody)
	if err != nil {
		return nil, fmt.Errorf("find projects without memories of their own: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
		}
	}()
	for rows.Next() {
		var p projectRef
		if err := rows.Scan(&p.id, &p.workspaceID); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// replacePlaceholder deletes p's placeholder by hand, since a migration runs with foreign keys off and no cascade fires;
// the 0130 trigger still takes its id out of auto plays' memory_ids.
func replacePlaceholder(ctx context.Context, tx *sql.Tx, p projectRef, seeds []memories.Seed, at int64) error {
	placeholders := `SELECT id FROM memories WHERE project_id = ? AND ` + placeholderOnly
	for _, stmt := range []string{
		`DELETE FROM attachments WHERE memory_id IN (` + placeholders + `)`,
		`DELETE FROM memory_versions WHERE memory_id IN (` + placeholders + `)`,
		`DELETE FROM memories WHERE id IN (` + placeholders + `)`,
	} {
		// hand-written: see projectsWithoutOwnMemories
		if _, err := tx.ExecContext(ctx, stmt, p.id, placeholderMemoryTitle, placeholderMemoryBody); err != nil {
			return fmt.Errorf("delete the placeholder memory: %w", err)
		}
	}
	for _, s := range seeds {
		id := ids.New()
		// hand-written: see projectsWithoutOwnMemories
		if _, err := tx.ExecContext(ctx, `INSERT INTO memories (id, workspace_id, project_id, kind, title, when_to_use, body, always_included, footer, version, created_by, created_at, updated_by, updated_at)
			VALUES (?, ?, ?, '', ?, ?, ?, ?, ?, 1, '', ?, '', ?)`,
			id, p.workspaceID, p.id, s.Title, s.WhenToUse, s.Body, boolToInt(s.AlwaysIncluded), boolToInt(s.Footer), at, at); err != nil {
			return fmt.Errorf("insert memory %q: %w", s.Title, err)
		}
		// hand-written: see projectsWithoutOwnMemories
		if _, err := tx.ExecContext(ctx, `INSERT INTO memory_versions (id, memory_id, version, title, when_to_use, body, always_included, editor_ids, created_at)
			VALUES (?, ?, 1, ?, ?, ?, ?, '[]', ?)`,
			ids.New(), id, s.Title, s.WhenToUse, s.Body, boolToInt(s.AlwaysIncluded), at); err != nil {
			return fmt.Errorf("insert the first version of memory %q: %w", s.Title, err)
		}
	}
	return nil
}
