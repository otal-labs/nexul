package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/plays"
)

// migratedPlay is a built-in play whose instructions a migration rewrote, with its newest kept version before that.
type migratedPlay struct {
	id, instructions string
	newest           sql.NullInt64
	newestBody       sql.NullString
}

// versionRewrittenBuiltinPlays adds the instructions a migration's script gave a built-in play as its newest version,
// the seeded default, sized and pruned as a save would, so its history and the version a run records end at the text it
// runs with.
func versionRewrittenBuiltinPlays(ctx context.Context, tx *sql.Tx) error {
	changed, err := rewrittenBuiltinPlays(ctx, tx)
	if err != nil {
		return err
	}
	for _, p := range changed {
		if err := addMigratedVersion(ctx, tx, p); err != nil {
			return fmt.Errorf("version play %s: %w", p.id, err)
		}
	}
	return nil
}

func rewrittenBuiltinPlays(ctx context.Context, tx *sql.Tx) (out []migratedPlay, err error) {
	// hand-written: a migration step reads the schema as of its own migration, which the generated queries don't
	rows, err := tx.QueryContext(ctx, `
SELECT p.id, p.instructions, v.version, v.body
FROM plays p
LEFT JOIN play_instruction_versions v ON v.play_id = p.id
    AND v.version = (SELECT MAX(n.version) FROM play_instruction_versions n WHERE n.play_id = p.id)
WHERE p.builtin_key IN ('fix-with-ai', 'to-tickets-via-ai', 'interview', 'interview-draft', 'audit', 'test-with-ai', 'clarify', 'update-doc')
  AND p.instructions IS NOT v.body
ORDER BY p.id`)
	if err != nil {
		return nil, fmt.Errorf("list migrated plays: %w", err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var p migratedPlay
		if err := rows.Scan(&p.id, &p.instructions, &p.newest, &p.newestBody); err != nil {
			return nil, fmt.Errorf("scan migrated play: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func addMigratedVersion(ctx context.Context, tx *sql.Tx, p migratedPlay) error {
	var size versionSize
	if p.newest.Valid {
		size = sizeOf(p.newestBody.String, p.instructions)
	}
	version := p.newest.Int64 + 1
	// hand-written: see rewrittenBuiltinPlays
	_, err := tx.ExecContext(ctx, `
INSERT INTO play_instruction_versions (play_id, version, body, editor_ids, via, builtin_default, words_added, words_removed, created_at)
VALUES (?, ?, ?, '[]', '', 1, ?, ?, CAST(strftime('%s', 'now') AS INTEGER))`,
		p.id, version, p.instructions, size.added, size.removed)
	if err != nil {
		return fmt.Errorf("insert version %d: %w", version, err)
	}
	// hand-written: see rewrittenBuiltinPlays
	res, err := tx.ExecContext(ctx, `DELETE FROM play_instruction_versions WHERE play_id = ? AND version <= ?`, p.id, version-plays.KeptInstructionVersions)
	if err != nil {
		return fmt.Errorf("prune versions: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return err
	}
	// hand-written: see rewrittenBuiltinPlays; the new oldest kept version has no predecessor to be sized against
	_, err = tx.ExecContext(ctx, `
UPDATE play_instruction_versions SET words_added = NULL, words_removed = NULL
WHERE play_id = ?1 AND version = (SELECT MIN(version) FROM play_instruction_versions WHERE play_id = ?1)`, p.id)
	if err != nil {
		return fmt.Errorf("clear the size of the oldest kept version: %w", err)
	}
	return nil
}
