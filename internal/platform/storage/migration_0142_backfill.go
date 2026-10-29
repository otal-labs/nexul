package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/docs/richtext"
)

// versionSizeTable names one history table as migration 0142 saw it: its item key column, the column a version's
// title is read from (” where the table has none), and the text its words are counted over. The SQL below is frozen
// to that schema, never to the generated queries.
type versionSizeTable struct {
	table, key, title string
	text              func(title, body string) string
}

var versionSizeTables = []versionSizeTable{
	{"doc_versions", "doc_id", "title", docVersionText},
	{"ticket_description_versions", "ticket_id", "''", func(_, body string) string { return richtext.PlainText(body) }},
	{"play_instruction_versions", "play_id", "''", func(_, body string) string { return body }},
}

// backfillVersionSizes fills the words added and removed of every kept version against its kept predecessor, by
// version number per item; the oldest kept version of an item stays null.
func backfillVersionSizes(ctx context.Context, tx *sql.Tx) error {
	for _, t := range versionSizeTables {
		keys, err := backfillKeys(ctx, tx, t)
		if err != nil {
			return err
		}
		for _, key := range keys {
			if err := backfillItem(ctx, tx, t, key); err != nil {
				return fmt.Errorf("size versions of %s %s: %w", t.table, key, err)
			}
		}
	}
	return nil
}

func backfillKeys(ctx context.Context, tx *sql.Tx, t versionSizeTable) (keys []string, err error) {
	// hand-written: a migration step reads the schema as of its own migration, which the generated queries don't
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT `+t.key+` FROM `+t.table)
	if err != nil {
		return nil, fmt.Errorf("list items of %s: %w", t.table, err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan item of %s: %w", t.table, err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

type backfillVersion struct {
	number int64
	text   string
}

func backfillItem(ctx context.Context, tx *sql.Tx, t versionSizeTable, key string) error {
	versions, err := backfillVersions(ctx, tx, t, key)
	if err != nil {
		return err
	}
	for i := 1; i < len(versions); i++ {
		size := sizeOf(versions[i-1].text, versions[i].text)
		// hand-written: see backfillKeys
		_, err := tx.ExecContext(ctx, `UPDATE `+t.table+` SET words_added = ?, words_removed = ? WHERE `+t.key+` = ? AND version = ?`,
			size.added, size.removed, key, versions[i].number)
		if err != nil {
			return fmt.Errorf("set size of version %d: %w", versions[i].number, err)
		}
	}
	return nil
}

func backfillVersions(ctx context.Context, tx *sql.Tx, t versionSizeTable, key string) (out []backfillVersion, err error) {
	// hand-written: see backfillKeys
	rows, err := tx.QueryContext(ctx, `SELECT version, `+t.title+`, body FROM `+t.table+` WHERE `+t.key+` = ? ORDER BY version`, key)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var v backfillVersion
		var title, body string
		if err := rows.Scan(&v.number, &title, &body); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		v.text = t.text(title, body)
		out = append(out, v)
	}
	return out, rows.Err()
}
