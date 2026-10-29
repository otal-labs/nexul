package storage

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/platform/wordcount"
)

// versionSizer reads and sizes the kept versions of one item (a doc, a ticket, a play) in its history table.
// A version's size is the words added and removed against the kept version before it; the oldest kept has none.
type versionSizer struct {
	numbers func(ctx context.Context) ([]int64, error)
	text    func(ctx context.Context, version int64) (string, error)
	set     func(ctx context.Context, version int64, size versionSize) error
}

type versionSize struct{ added, removed sql.NullInt64 }

// nullableInt is a stored size as the pointer the wire carries: nil for none.
func nullableInt(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// restoredFromOf stores a restore's source version, and none for an ordinary save.
func restoredFromOf(version int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(version), Valid: version > 0}
}

func sizeOf(before, after string) versionSize {
	added, removed := wordcount.Changed(before, after)
	return versionSize{
		added:   sql.NullInt64{Int64: int64(added), Valid: true},
		removed: sql.NullInt64{Int64: int64(removed), Valid: true},
	}
}

// docVersionText is what a doc version's words are counted over: its title, then its body without markup.
func docVersionText(title, body string) string {
	return title + " " + richtext.PlainText(body)
}

func docSizer(q *sqlcgen.Queries, docID string) versionSizer {
	return versionSizer{
		numbers: func(ctx context.Context) ([]int64, error) { return q.ListDocVersionNumbers(ctx, docID) },
		text: func(ctx context.Context, version int64) (string, error) {
			row, err := q.GetDocVersion(ctx, sqlcgen.GetDocVersionParams{DocID: docID, Version: version})
			return docVersionText(row.Title, row.Body), err
		},
		set: func(ctx context.Context, version int64, size versionSize) error {
			return q.SetDocVersionSize(ctx, sqlcgen.SetDocVersionSizeParams{
				WordsAdded: size.added, WordsRemoved: size.removed, DocID: docID, Version: version,
			})
		},
	}
}

func memorySizer(q *sqlcgen.Queries, memoryID string) versionSizer {
	return versionSizer{
		numbers: func(ctx context.Context) ([]int64, error) { return q.ListMemoryVersionNumbers(ctx, memoryID) },
		text: func(ctx context.Context, version int64) (string, error) {
			row, err := q.GetMemoryVersion(ctx, sqlcgen.GetMemoryVersionParams{MemoryID: memoryID, Version: version})
			return docVersionText(row.Title, row.Body), err
		},
		set: func(ctx context.Context, version int64, size versionSize) error {
			return q.SetMemoryVersionSize(ctx, sqlcgen.SetMemoryVersionSizeParams{
				WordsAdded: size.added, WordsRemoved: size.removed, MemoryID: memoryID, Version: version,
			})
		},
	}
}

func ticketSizer(q *sqlcgen.Queries, ticketID string) versionSizer {
	return versionSizer{
		numbers: func(ctx context.Context) ([]int64, error) {
			return q.ListTicketDescriptionVersionNumbers(ctx, ticketID)
		},
		text: func(ctx context.Context, version int64) (string, error) {
			row, err := q.GetTicketDescriptionVersion(ctx, sqlcgen.GetTicketDescriptionVersionParams{TicketID: ticketID, Version: version})
			return richtext.PlainText(row.Body), err
		},
		set: func(ctx context.Context, version int64, size versionSize) error {
			return q.SetTicketDescriptionVersionSize(ctx, sqlcgen.SetTicketDescriptionVersionSizeParams{
				WordsAdded: size.added, WordsRemoved: size.removed, TicketID: ticketID, Version: version,
			})
		},
	}
}

func playSizer(q *sqlcgen.Queries, playID string) versionSizer {
	return versionSizer{
		numbers: func(ctx context.Context) ([]int64, error) { return q.ListPlayInstructionVersionNumbers(ctx, playID) },
		text: func(ctx context.Context, version int64) (string, error) {
			row, err := q.GetPlayInstructionVersion(ctx, sqlcgen.GetPlayInstructionVersionParams{PlayID: playID, Version: version})
			return row.Body, err
		},
		set: func(ctx context.Context, version int64, size versionSize) error {
			return q.SetPlayInstructionVersionSize(ctx, sqlcgen.SetPlayInstructionVersionSizeParams{
				WordsAdded: size.added, WordsRemoved: size.removed, PlayID: playID, Version: version,
			})
		},
	}
}

// againstNewest sizes a version about to be inserted with text against the item's newest kept version; the first
// version of an item has no size.
func (s versionSizer) againstNewest(ctx context.Context, text string) (versionSize, error) {
	kept, err := s.numbers(ctx)
	if err != nil {
		return versionSize{}, fmt.Errorf("list kept versions: %w", err)
	}
	if len(kept) == 0 {
		return versionSize{}, nil
	}
	newest, err := s.text(ctx, kept[len(kept)-1])
	if err != nil {
		return versionSize{}, fmt.Errorf("read newest kept version: %w", err)
	}
	return sizeOf(newest, text), nil
}

// pruned runs drop, then resizes every kept version whose predecessor it removed and clears the new oldest.
func (s versionSizer) pruned(ctx context.Context, drop func() error) error {
	before, err := s.numbers(ctx)
	if err != nil {
		return fmt.Errorf("list kept versions: %w", err)
	}
	if err := drop(); err != nil {
		return err
	}
	after, err := s.numbers(ctx)
	if err != nil {
		return fmt.Errorf("list kept versions: %w", err)
	}
	if len(after) == len(before) {
		return nil
	}
	return s.resize(ctx, before, after)
}

// resize sizes each version of after whose predecessor differs from its predecessor in before.
func (s versionSizer) resize(ctx context.Context, before, after []int64) error {
	var cached string
	var cachedVersion int64
	for i, version := range after {
		var predecessor int64
		if i > 0 {
			predecessor = after[i-1]
		}
		if predecessorIn(before, version) == predecessor {
			continue
		}
		if predecessor == 0 {
			if err := s.set(ctx, version, versionSize{}); err != nil {
				return fmt.Errorf("clear size of version %d: %w", version, err)
			}
			continue
		}
		if cachedVersion != predecessor {
			var err error
			if cached, err = s.text(ctx, predecessor); err != nil {
				return fmt.Errorf("read version %d: %w", predecessor, err)
			}
		}
		text, err := s.text(ctx, version)
		if err != nil {
			return fmt.Errorf("read version %d: %w", version, err)
		}
		if err := s.set(ctx, version, sizeOf(cached, text)); err != nil {
			return fmt.Errorf("size version %d: %w", version, err)
		}
		cached, cachedVersion = text, version
	}
	return nil
}

// predecessorIn is the version just below version in the ascending kept, or 0 when it was the oldest or is absent.
func predecessorIn(kept []int64, version int64) int64 {
	i, found := slices.BinarySearch(kept, version)
	if !found || i == 0 {
		return 0
	}
	return kept[i-1]
}
